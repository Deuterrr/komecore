package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type mockRepository struct {
	events          []Event
	completedIDs    []uuid.UUID
	failedIDs       []uuid.UUID
	lastRetryCounts map[uuid.UUID]int
	lastErrors      map[uuid.UUID]string
	lastScheduled   map[uuid.UUID]time.Time
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		lastRetryCounts: make(map[uuid.UUID]int),
		lastErrors:      make(map[uuid.UUID]string),
		lastScheduled:   make(map[uuid.UUID]time.Time),
	}
}

func (m *mockRepository) Enqueue(_ context.Context, _ transaction.Executor, eventType string, payload any) error {
	bytes, _ := json.Marshal(payload)
	m.events = append(m.events, Event{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   bytes,
		Status:    StatusPending,
	})
	return nil
}

func (m *mockRepository) FetchPending(_ context.Context, _ transaction.Executor, limit int) ([]Event, error) {
	if len(m.events) > limit {
		return m.events[:limit], nil
	}
	return m.events, nil
}

func (m *mockRepository) MarkCompleted(_ context.Context, _ transaction.Executor, id uuid.UUID, _ time.Time) error {
	m.completedIDs = append(m.completedIDs, id)
	return nil
}

func (m *mockRepository) MarkFailed(_ context.Context, _ transaction.Executor, id uuid.UUID, retryCount int, nextScheduledAt time.Time, lastError string) error {
	m.failedIDs = append(m.failedIDs, id)
	m.lastRetryCounts[id] = retryCount
	m.lastErrors[id] = lastError
	m.lastScheduled[id] = nextScheduledAt
	return nil
}

type mockDispatcher struct {
	dispatched []Event
	errMap     map[uuid.UUID]error
}

func (d *mockDispatcher) Dispatch(_ context.Context, event Event) error {
	d.dispatched = append(d.dispatched, event)
	if err, exists := d.errMap[event.ID]; exists {
		return err
	}
	return nil
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		retry    int
		expected time.Duration
	}{
		{retry: 0, expected: 10 * time.Second},  // 2^0 * 10s = 10s
		{retry: 1, expected: 20 * time.Second},  // 2^1 * 10s = 20s
		{retry: 2, expected: 40 * time.Second},  // 2^2 * 10s = 40s
		{retry: 3, expected: 80 * time.Second},  // 2^3 * 10s = 80s
		{retry: 4, expected: 160 * time.Second}, // 2^4 * 10s = 160s
		{retry: 5, expected: 320 * time.Second}, // 2^5 * 10s = 320s
	}

	for _, tc := range tests {
		actual := CalculateBackoff(tc.retry)
		if actual != tc.expected {
			t.Errorf("CalculateBackoff(%d) = %v, expected %v", tc.retry, actual, tc.expected)
		}
	}
}

func TestWorker_ProcessBatch_Success(t *testing.T) {
	repo := newMockRepository()
	disp := &mockDispatcher{errMap: make(map[uuid.UUID]error)}

	eventID := uuid.New()
	repo.events = []Event{
		{
			ID:        eventID,
			EventType: EventOrderCreated,
			Payload:   json.RawMessage(`{"order_number":"ORD-001"}`),
			Status:    StatusPending,
		},
	}

	worker := NewWorker(
		repo,
		disp,
		&transaction.NoopExecutor{},
		&transaction.NoopTransactor{},
		nil,
		1*time.Second,
		10,
	)

	count, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 processed event, got %d", count)
	}

	if len(disp.dispatched) != 1 || disp.dispatched[0].ID != eventID {
		t.Errorf("expected event %s to be dispatched", eventID)
	}

	if len(repo.completedIDs) != 1 || repo.completedIDs[0] != eventID {
		t.Errorf("expected event %s to be marked completed", eventID)
	}
}

func TestWorker_ProcessBatch_FailureAndExponentialBackoff(t *testing.T) {
	repo := newMockRepository()
	disp := &mockDispatcher{errMap: make(map[uuid.UUID]error)}

	eventID := uuid.New()
	repo.events = []Event{
		{
			ID:         eventID,
			EventType:  EventPaymentSettled,
			Payload:    json.RawMessage(`{"order_number":"ORD-002"}`),
			Status:     StatusPending,
			RetryCount: 1,
			MaxRetries: 5,
		},
	}

	smtpErr := errors.New("connection timeout to smtp.gmail.com")
	disp.errMap[eventID] = smtpErr

	worker := NewWorker(
		repo,
		disp,
		&transaction.NoopExecutor{},
		&transaction.NoopTransactor{},
		nil,
		1*time.Second,
		10,
	)

	before := time.Now()
	count, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 event batch iteration, got %d", count)
	}

	if len(repo.completedIDs) != 0 {
		t.Errorf("event should not be completed on error")
	}

	if len(repo.failedIDs) != 1 || repo.failedIDs[0] != eventID {
		t.Fatalf("expected event %s to be marked failed/rescheduled", eventID)
	}

	if repo.lastRetryCounts[eventID] != 2 {
		t.Errorf("expected retry count 2, got %d", repo.lastRetryCounts[eventID])
	}

	if repo.lastErrors[eventID] != smtpErr.Error() {
		t.Errorf("expected error %q, got %q", smtpErr.Error(), repo.lastErrors[eventID])
	}

	// Verify backoff for retry = 2: 2^2 * 10s = 40s
	scheduled := repo.lastScheduled[eventID]
	diff := scheduled.Sub(before)
	if diff < 39*time.Second || diff > 45*time.Second {
		t.Errorf("expected backoff ~40s, got %v", diff)
	}
}

func TestWorker_Start_GracefulStop(t *testing.T) {
	repo := newMockRepository()
	disp := &mockDispatcher{errMap: make(map[uuid.UUID]error)}

	worker := NewWorker(
		repo,
		disp,
		&transaction.NoopExecutor{},
		&transaction.NoopTransactor{},
		nil,
		50*time.Millisecond,
		10,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		worker.Start(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Clean exit
	case <-time.After(1 * time.Second):
		t.Fatal("worker did not stop cleanly on context cancel")
	}
}
