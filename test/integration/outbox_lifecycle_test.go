//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"komecore/internal/infra/outbox"
	applogger "komecore/pkg/logger"
	"komecore/test/testutil/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockIntegrationDispatcher struct {
	mu         sync.Mutex
	dispatched []outbox.Event
	fail       bool
}

func (m *mockIntegrationDispatcher) Dispatch(ctx context.Context, event outbox.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("simulated SMTP delivery failure")
	}
	m.dispatched = append(m.dispatched, event)
	return nil
}

func TestIntegration_Outbox_WorkerLifecycle_And_Backoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tdb, err := testdb.NewTestDB(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err)
	defer func() {
		_ = tdb.Close(ctx)
	}()

	err = tdb.ApplyMigrations()
	require.NoError(t, err)
	defer func() {
		_ = tdb.Truncate(ctx)
	}()

	repo := outbox.NewOutboxRepository()
	dispatcher := &mockIntegrationDispatcher{}
	log := applogger.NewZapLogger("test")

	worker := outbox.NewWorker(
		repo,
		dispatcher,
		tdb.Executor,
		tdb.Transactor,
		log,
		5*time.Second,
		10,
	)

	// 1. Success Flow: Insert pending event and process batch
	eventID := uuid.New()
	orderID := uuid.New()
	payload := outbox.OrderCreatedPayload{
		OrderID:       orderID,
		OrderNumber:   "ORD-OUTBOX-101",
		CustomerEmail: "buyer@example.com",
		CustomerName:  "Test Buyer",
		Total:         300000,
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	now := time.Now().UTC()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, retry_count, max_retries, scheduled_at, created_at)
		VALUES ($1, $2, $3, 'pending', 0, 5, $4, $4);
	`, eventID, outbox.EventOrderCreated, payloadBytes, now)
	require.NoError(t, err)

	processed, err := worker.ProcessBatch(ctx)
	require.NoError(t, err, "batch processing must succeed")
	assert.Equal(t, 1, processed, "must process exactly 1 event")

	// Verify database record transitioned to completed
	var status string
	var processedAt *time.Time
	var lastErr *string
	err = tdb.Pool.QueryRow(ctx, "SELECT status, processed_at, last_error FROM outbox_events WHERE id = $1;", eventID).Scan(&status, &processedAt, &lastErr)
	require.NoError(t, err)
	assert.Equal(t, "completed", status)
	assert.NotNil(t, processedAt)
	assert.Nil(t, lastErr)

	// 2. Failure & Backoff Flow: Simulate dispatcher error
	failEventID := uuid.New()
	dispatcher.fail = true // trigger failure

	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, retry_count, max_retries, scheduled_at, created_at)
		VALUES ($1, $2, $3, 'pending', 0, 5, $4, $4);
	`, failEventID, outbox.EventOrderCreated, payloadBytes, now)
	require.NoError(t, err)

	processedFail, err := worker.ProcessBatch(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, processedFail, "failed event counts as processed attempt")

	// Verify database record has retry_count=1, last_error populated, status=pending, scheduled_at rescheduled
	var failStatus string
	var retryCount int
	var scheduledAt time.Time
	var recordedError *string
	err = tdb.Pool.QueryRow(ctx, "SELECT status, retry_count, scheduled_at, last_error FROM outbox_events WHERE id = $1;", failEventID).Scan(
		&failStatus, &retryCount, &scheduledAt, &recordedError,
	)
	require.NoError(t, err)
	assert.Equal(t, "pending", failStatus)
	assert.Equal(t, 1, retryCount)
	assert.NotNil(t, recordedError)
	assert.Contains(t, *recordedError, "simulated SMTP delivery failure")

	// Verify exponential backoff: scheduled_at >= now + 20s (2^1 * 10s)
	expectedMinScheduled := now.Add(18 * time.Second) // allow slight clock tolerance
	assert.True(t, scheduledAt.After(expectedMinScheduled), "scheduled_at (%v) must be >= 20s backoff", scheduledAt)

	// Clean up failEventID so its 20-second backoff doesn't re-enter pending queue during subsequent steps
	_, err = tdb.Pool.Exec(ctx, "DELETE FROM outbox_events WHERE id = $1;", failEventID)
	require.NoError(t, err)

	// 3. Max Retries Exhaustion: simulate retry_count = 4 reaching limit
	maxEventID := uuid.New()
	step3Now := time.Now().UTC()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, retry_count, max_retries, scheduled_at, created_at)
		VALUES ($1, $2, $3, 'pending', 4, 5, $4, $4);
	`, maxEventID, outbox.EventOrderCreated, payloadBytes, step3Now)
	require.NoError(t, err)

	_, err = worker.ProcessBatch(ctx)
	require.NoError(t, err)

	var maxStatus string
	var maxRetriesRecorded int
	err = tdb.Pool.QueryRow(ctx, "SELECT status, retry_count FROM outbox_events WHERE id = $1;", maxEventID).Scan(&maxStatus, &maxRetriesRecorded)
	require.NoError(t, err)
	assert.Equal(t, "failed", maxStatus, "event must transition to failed after reaching max retries")
	assert.Equal(t, 5, maxRetriesRecorded)

	// Clean up maxEventID after verification
	_, err = tdb.Pool.Exec(ctx, "DELETE FROM outbox_events WHERE id = $1;", maxEventID)
	require.NoError(t, err)

	// 4. Atomic Claim Verification (MC-C2): FetchPending transitions status to 'processing'
	claimEventID := uuid.New()
	claimNow := time.Now().UTC()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, retry_count, max_retries, scheduled_at, created_at)
		VALUES ($1, $2, $3, 'pending', 0, 5, $4, $4);
	`, claimEventID, outbox.EventOrderCreated, payloadBytes, claimNow)
	require.NoError(t, err)

	claimedEvents, err := repo.FetchPending(ctx, tdb.Executor, 10)
	require.NoError(t, err)
	assert.Len(t, claimedEvents, 1)
	assert.Equal(t, claimEventID, claimedEvents[0].ID)
	assert.Equal(t, outbox.StatusProcessing, claimedEvents[0].Status)

	// Second fetch must return 0 events because the row is now in 'processing' status
	secondFetch, err := repo.FetchPending(ctx, tdb.Executor, 10)
	require.NoError(t, err)
	assert.Empty(t, secondFetch, "concurrent or subsequent fetch must skip claimed processing event")
}
