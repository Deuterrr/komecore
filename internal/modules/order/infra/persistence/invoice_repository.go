package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InvoiceRepository struct{}

func NewInvoiceRepository() *InvoiceRepository {
	return &InvoiceRepository{}
}

func (r *InvoiceRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.Invoice, error) {
	query := `
		SELECT
			id,
			number,
			order_id,
			status,
			subtotal,
			shipping_fee,
			total,
			issued_at,
			created_at
		FROM
			invoices
		WHERE
			id = $1
		LIMIT 1
	`

	var invoice domain.Invoice
	err := exec.QueryRow(ctx, query, id).Scan(
		&invoice.ID,
		&invoice.Number,
		&invoice.OrderID,
		&invoice.Status,
		&invoice.Subtotal,
		&invoice.ShippingFee,
		&invoice.Total,
		&invoice.IssuedAt,
		&invoice.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query invoice by id failed: %w", err)
	}

	return &invoice, nil
}

func (r *InvoiceRepository) GetByOrderID(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) (*domain.Invoice, error) {
	query := `
		SELECT
			id,
			number,
			order_id,
			status,
			subtotal,
			shipping_fee,
			total,
			issued_at,
			created_at
		FROM
			invoices
		WHERE
			order_id = $1
		LIMIT 1
	`

	var invoice domain.Invoice
	err := exec.QueryRow(ctx, query, orderID).Scan(
		&invoice.ID,
		&invoice.Number,
		&invoice.OrderID,
		&invoice.Status,
		&invoice.Subtotal,
		&invoice.ShippingFee,
		&invoice.Total,
		&invoice.IssuedAt,
		&invoice.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query invoice by order id failed: %w", err)
	}

	return &invoice, nil
}

func (r *InvoiceRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	invoice domain.Invoice,
) error {
	query := `
		INSERT INTO invoices (
			id,
			number,
			order_id,
			status,
			subtotal,
			shipping_fee,
			total,
			issued_at,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id)
		DO UPDATE SET
			number = EXCLUDED.number,
			order_id = EXCLUDED.order_id,
			status = EXCLUDED.status,
			subtotal = EXCLUDED.subtotal,
			shipping_fee = EXCLUDED.shipping_fee,
			total = EXCLUDED.total,
			issued_at = EXCLUDED.issued_at
	`

	_, err := exec.Exec(ctx, query,
		invoice.ID,
		invoice.Number,
		invoice.OrderID,
		invoice.Status,
		invoice.Subtotal,
		invoice.ShippingFee,
		invoice.Total,
		invoice.IssuedAt,
		invoice.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to save invoice: %w", err)
	}

	return nil
}
