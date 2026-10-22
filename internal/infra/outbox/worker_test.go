package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type mockRepository struct {
	mu              sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
	bytes, _ := json.Marshal(payload)
	m.events = append(m.events, Event{
		ID:          uuid.New(),
		EventType:   eventType,
		Payload:     bytes,
		Status:      StatusPending,
		ScheduledAt: time.Now(),
	})
	return nil
}

func (m *mockRepository) FetchPending(_ context.Context, _ transaction.Executor, limit int) ([]Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	staleThreshold := now.Add(-10 * time.Minute)

	var claimed []Event
	for i := range m.events {
		// Match status='pending' (scheduled_at <= now) or stale status='processing' (scheduled_at <= now - 10m)
		isPending := m.events[i].Status == StatusPending && (m.events[i].ScheduledAt.IsZero() || !m.events[i].ScheduledAt.After(now))
		isStaleProcessing := m.events[i].Status == StatusProcessing && (!m.events[i].ScheduledAt.IsZero() && !m.events[i].ScheduledAt.After(staleThreshold))

		if isPending || isStaleProcessing {
			m.events[i].Status = StatusProcessing
			m.events[i].ScheduledAt = now
			claimed = append(claimed, m.events[i])
			if len(claimed) >= limit {
				break
			}
		}
	}
	return claimed, nil
}

func (m *mockRepository) MarkCompleted(_ context.Context, _ transaction.Executor, id uuid.UUID, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completedIDs = append(m.completedIDs, id)
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].Status = StatusCompleted
			break
		}
	}
	return nil
}

func (m *mockRepository) MarkFailed(_ context.Context, _ transaction.Executor, id uuid.UUID, retryCount int, nextScheduledAt time.Time, lastError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedIDs = append(m.failedIDs, id)
	m.lastRetryCounts[id] = retryCount
	m.lastErrors[id] = lastError
	m.lastScheduled[id] = nextScheduledAt
	for i := range m.events {
		if m.events[i].ID == id {
			if retryCount >= 5 {
				m.events[i].Status = StatusFailed
			} else {
				m.events[i].Status = StatusPending
			}
			m.events[i].RetryCount = retryCount
			m.events[i].ScheduledAt = nextScheduledAt
			break
		}
	}
	return nil
}

type mockDispatcher struct {
	mu         sync.Mutex
	dispatched []Event
	errMap     map[uuid.UUID]error
	delay      time.Duration
}

func (d *mockDispatcher) Dispatch(_ context.Context, event Event) error {
	if d.delay > 0 {
		time.Sleep(d.delay)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
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

func TestWorker_MultiWorker_NoDuplicateDispatches(t *testing.T) {
	const numEvents = 30
	const numWorkers = 3

	repo := newMockRepository()
	disp := &mockDispatcher{
		errMap: make(map[uuid.UUID]error),
		delay:  2 * time.Millisecond, // Simulate network I/O
	}

	for i := 0; i < numEvents; i++ {
		_ = repo.Enqueue(context.Background(), &transaction.NoopExecutor{}, EventOrderCreated, map[string]string{
			"order": fmt.Sprintf("ORD-%03d", i),
		})
	}

	// Track deliveries across all workers to ensure zero duplicate dispatches
	var (
		deliveryMu  sync.Mutex
		deliveryMap = make(map[uuid.UUID]int)
		duplicates  atomic.Int64
	)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			worker := NewWorker(
				repo,
				disp,
				&transaction.NoopExecutor{},
				&transaction.NoopTransactor{},
				nil,
				10*time.Millisecond,
				10, // batch size 10
			)

			// Process batches until no pending events remain
			for {
				count, err := worker.ProcessBatch(context.Background())
				if err != nil {
					t.Errorf("worker %d process batch error: %v", workerID, err)
					return
				}
				if count == 0 {
					return
				}
			}
		}(w)
	}

	wg.Wait()

	// Verify every dispatched event was received exactly once
	disp.mu.Lock()
	for _, event := range disp.dispatched {
		deliveryMu.Lock()
		deliveryMap[event.ID]++
		if deliveryMap[event.ID] > 1 {
			duplicates.Add(1)
		}
		deliveryMu.Unlock()
	}
	totalDispatched := len(disp.dispatched)
	disp.mu.Unlock()

	if dup := duplicates.Load(); dup > 0 {
		t.Fatalf("detected %d duplicate event dispatches across %d workers!", dup, numWorkers)
	}

	if totalDispatched != numEvents {
		t.Fatalf("expected %d total dispatches, got %d", numEvents, totalDispatched)
	}

	repo.mu.Lock()
	completedCount := len(repo.completedIDs)
	repo.mu.Unlock()

	if completedCount != numEvents {
		t.Fatalf("expected %d completed events in repository, got %d", numEvents, completedCount)
	}
}

func TestWorker_ProcessBatch_StaleProcessingReclaimed(t *testing.T) {
	repo := newMockRepository()
	disp := &mockDispatcher{errMap: make(map[uuid.UUID]error)}

	staleEventID := uuid.New()
	// Event was claimed 15 minutes ago by a crashed worker
	repo.events = []Event{
		{
			ID:          staleEventID,
			EventType:   EventOrderCreated,
			Payload:     json.RawMessage(`{"order_number":"ORD-CRASH-001"}`),
			Status:      StatusProcessing,
			ScheduledAt: time.Now().Add(-15 * time.Minute),
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
		t.Fatalf("expected 1 stale event reclaimed and processed, got %d", count)
	}

	if len(disp.dispatched) != 1 || disp.dispatched[0].ID != staleEventID {
		t.Fatalf("expected stale event %s to be dispatched", staleEventID)
	}

	if len(repo.completedIDs) != 1 || repo.completedIDs[0] != staleEventID {
		t.Fatalf("expected stale event %s to be marked completed", staleEventID)
	}
}
