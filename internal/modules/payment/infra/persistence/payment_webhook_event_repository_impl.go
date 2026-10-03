package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/domain"
	"komecore/internal/modules/payment/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type paymentWebhookEventRepositoryImpl struct{}

func NewPaymentWebhookEventRepositoryImpl() repository.PaymentWebhookEventRepository {
	return &paymentWebhookEventRepositoryImpl{}
}

// Upsert inserts a new webhook event row.
// On conflict (gateway_order_id, transaction_status) the existing row is returned
// unchanged, allowing the caller to inspect its status before deciding
// whether to re-process.
func (r *paymentWebhookEventRepositoryImpl) Upsert(
	ctx context.Context,
	exec transaction.Executor,
	event domain.PaymentWebhookEvent,
) (*domain.PaymentWebhookEvent, error) {
	// INSERT ... ON CONFLICT DO NOTHING, then SELECT to return the
	// authoritative row (either the newly inserted one or the pre-existing one).
	insertQuery := `
		INSERT INTO payment_webhook_events (
			id,
			gateway_order_id,
			transaction_status,
			payload,
			status,
			received_at
		)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (gateway_order_id, transaction_status)
		DO NOTHING
	`

	_, err := exec.Exec(ctx, insertQuery,
		event.ID,
		event.OrderID,
		event.TransactionStatus,
		event.Payload,
		event.Status,
		event.ReceivedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert payment webhook event failed: %w", err)
	}

	// Fetch the canonical row (may be the just-inserted row or the
	// pre-existing conflicting row).
	selectQuery := `
		SELECT
			id,
			gateway_order_id,
			transaction_status,
			payload,
			status,
			error,
			received_at,
			processed_at
		FROM payment_webhook_events
		WHERE gateway_order_id = $1
		  AND transaction_status = $2
	`

	return r.scanWebhookEvent(
		exec.QueryRow(ctx, selectQuery, event.OrderID, event.TransactionStatus),
	)
}

// MarkProcessed sets status = 'processed' and stamps processed_at = NOW().
func (r *paymentWebhookEventRepositoryImpl) MarkProcessed(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE payment_webhook_events
		SET
			status = 'processed',
			processed_at = $2
		WHERE id = $1
	`

	_, err := exec.Exec(ctx, query, id, appclock.Now())
	if err != nil {
		return fmt.Errorf("mark webhook event processed failed: %w", err)
	}

	return nil
}

// MarkFailed sets status = 'failed' and records the error string.
func (r *paymentWebhookEventRepositoryImpl) MarkFailed(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	errMsg string,
) error {
	query := `
		UPDATE payment_webhook_events
		SET
			status = 'failed',
			error = $2
		WHERE id = $1
	`

	_, err := exec.Exec(ctx, query, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark webhook event failed: %w", err)
	}

	return nil
}

func (r *paymentWebhookEventRepositoryImpl) scanWebhookEvent(
	row transaction.Row,
) (*domain.PaymentWebhookEvent, error) {
	var e domain.PaymentWebhookEvent
	err := row.Scan(
		&e.ID,
		&e.OrderID,
		&e.TransactionStatus,
		&e.Payload,
		&e.Status,
		&e.Error,
		&e.ReceivedAt,
		&e.ProcessedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan payment webhook event failed: %w", err)
	}
	return &e, nil
}
