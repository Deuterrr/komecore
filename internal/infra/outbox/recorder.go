package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

// Recorder defines the contract to record outbox events inside an atomic transaction.
type Recorder interface {
	Enqueue(ctx context.Context, exec transaction.Executor, eventType string, payload any) error
}

// Repository defines complete persistence operations for the outbox worker.
type Repository interface {
	Recorder
	FetchPending(ctx context.Context, exec transaction.Executor, limit int) ([]Event, error)
	MarkCompleted(ctx context.Context, exec transaction.Executor, id uuid.UUID, processedAt time.Time) error
	MarkFailed(ctx context.Context, exec transaction.Executor, id uuid.UUID, retryCount int, nextScheduledAt time.Time, lastError string) error
}

type OutboxRepository struct{}

func NewOutboxRepository() *OutboxRepository {
	return &OutboxRepository{}
}

func (r *OutboxRepository) Enqueue(
	ctx context.Context,
	exec transaction.Executor,
	eventType string,
	payload any,
) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox event payload: %w", err)
	}

	id := uuid.New()
	now := appclock.Now()

	query := `
		INSERT INTO outbox_events (
			id,
			event_type,
			payload,
			status,
			retry_count,
			max_retries,
			scheduled_at,
			created_at
		) VALUES ($1, $2, $3, 'pending', 0, 5, $4, $4)
	`

	_, err = exec.Exec(ctx, query, id, eventType, payloadBytes, now)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

func (r *OutboxRepository) FetchPending(
	ctx context.Context,
	exec transaction.Executor,
	limit int,
) ([]Event, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			id,
			event_type,
			payload,
			status,
			retry_count,
			max_retries,
			last_error,
			scheduled_at,
			processed_at,
			created_at
		FROM outbox_events
		WHERE status = 'pending'
		  AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	now := appclock.Now()
	rows, err := exec.Query(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending outbox events: %w", err)
	}
	if rows == nil {
		return []Event{}, nil
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var (
			e          Event
			statusStr  string
			rawPayload []byte
		)

		err := rows.Scan(
			&e.ID,
			&e.EventType,
			&rawPayload,
			&statusStr,
			&e.RetryCount,
			&e.MaxRetries,
			&e.LastError,
			&e.ScheduledAt,
			&e.ProcessedAt,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}

		e.Status = Status(statusStr)
		e.Payload = json.RawMessage(rawPayload)
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error while fetching pending outbox events: %w", err)
	}

	if events == nil {
		events = []Event{}
	}

	return events, nil
}

func (r *OutboxRepository) MarkCompleted(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	processedAt time.Time,
) error {
	query := `
		UPDATE outbox_events
		SET status = 'completed',
		    processed_at = $2,
		    last_error = NULL
		WHERE id = $1
	`

	_, err := exec.Exec(ctx, query, id, processedAt)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as completed: %w", err)
	}

	return nil
}

func (r *OutboxRepository) MarkFailed(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	retryCount int,
	nextScheduledAt time.Time,
	lastError string,
) error {
	const maxRetries = 5

	if retryCount >= maxRetries {
		query := `
			UPDATE outbox_events
			SET status = 'failed',
			    retry_count = $2,
			    last_error = $3,
			    processed_at = $4
			WHERE id = $1
		`
		now := appclock.Now()
		_, err := exec.Exec(ctx, query, id, retryCount, lastError, now)
		if err != nil {
			return fmt.Errorf("failed to mark outbox event as permanently failed: %w", err)
		}
		return nil
	}

	query := `
		UPDATE outbox_events
		SET status = 'pending',
		    retry_count = $2,
		    last_error = $3,
		    scheduled_at = $4
		WHERE id = $1
	`
	_, err := exec.Exec(ctx, query, id, retryCount, lastError, nextScheduledAt)
	if err != nil {
		return fmt.Errorf("failed to update outbox event retry schedule: %w", err)
	}

	return nil
}
