package outbox

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

func TestOutboxRepository_Enqueue(t *testing.T) {
	repo := NewOutboxRepository()
	exec := &transaction.NoopExecutor{}

	payload := OrderCreatedPayload{
		OrderID:       uuid.New(),
		OrderNumber:   "ORD-TEST-001",
		CustomerEmail: "test@example.com",
		CustomerName:  "Test Customer",
		Total:         100000,
	}

	err := repo.Enqueue(context.Background(), exec, EventOrderCreated, payload)
	if err != nil {
		t.Fatalf("failed to enqueue outbox event: %v", err)
	}
}

func TestOutboxRepository_MarkCompleted(t *testing.T) {
	repo := NewOutboxRepository()
	exec := &transaction.NoopExecutor{}

	err := repo.MarkCompleted(context.Background(), exec, uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("failed to mark event completed: %v", err)
	}
}

func TestOutboxRepository_MarkFailed(t *testing.T) {
	repo := NewOutboxRepository()
	exec := &transaction.NoopExecutor{}

	// Transient failure (retry < 5)
	err := repo.MarkFailed(context.Background(), exec, uuid.New(), 2, time.Now().Add(40*time.Second), "transient error")
	if err != nil {
		t.Fatalf("failed to mark event failed: %v", err)
	}

	// Permanent failure (retry >= 5)
	err = repo.MarkFailed(context.Background(), exec, uuid.New(), 5, time.Now(), "permanent error")
	if err != nil {
		t.Fatalf("failed to mark event permanently failed: %v", err)
	}
}
