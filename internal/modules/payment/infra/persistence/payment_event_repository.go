package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentEventRepository struct{}

func NewPaymentEventRepository() *PaymentEventRepository {
	return &PaymentEventRepository{}
}

func (r *PaymentEventRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.PaymentEvent, error) {
	query := `
		SELECT
			id
			payment_id
			event_name
			payload
			created_at
		FROM
			payment_events
		WHERE id = $1
		LIMIT 1
	`

	var event domain.PaymentEvent
	err := exec.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.PaymentID,
		&event.EventName,
		&event.Payload,
		&event.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment event by id failed: %w", err)
	}

	return &event, nil
}

func (r *PaymentEventRepository) ListByPaymentID(
	ctx context.Context,
	exec transaction.Executor,
	paymentID uuid.UUID,
) ([]domain.PaymentEvent, error) {
	query := `
		SELECT
			id
			payment_id
			event_name
			payload
			created_at
		FROM
			payment_events
		WHERE payment_id = $1
	`

	rows, err := exec.Query(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("query payment events failed: %w", err)
	}
	defer rows.Close()

	var events []domain.PaymentEvent
	for rows.Next() {
		var event domain.PaymentEvent
		err := rows.Scan(
			&event.ID,
			&event.PaymentID,
			&event.EventName,
			&event.Payload,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping payment event model to domain failed: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment events failed: %w", err)
	}

	return events, nil
}

func (r *PaymentEventRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	event domain.PaymentEvent,
) error {
	query := `
		INSERT INTO payment_events (
			id,
			payment_id,
			event_name,
			payload,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5)
	`

	_, err := exec.Exec(ctx, query,
		event.ID,
		event.PaymentID,
		event.EventName,
		event.Payload,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to save payment event failed: %w", err)
	}

	return nil
}
