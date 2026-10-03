package persistence

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/domain"
	"komecore/internal/modules/order/repository"

	"github.com/google/uuid"
)

type invoiceItemRepositoryImpl struct{}

func NewInvoiceItemRepositoryImpl() repository.InvoiceItemRepository {
	return &invoiceItemRepositoryImpl{}
}

func (r *invoiceItemRepositoryImpl) ListByInvoiceID(
	ctx context.Context,
	exec transaction.Executor,
	invoiceID uuid.UUID,
) ([]domain.InvoiceItem, error) {
	query := `
		SELECT
			id,
			invoice_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total
		FROM
			invoice_items
		WHERE
			invoice_id = $1
	`

	rows, err := exec.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("query invoice items by invoice id failed: %w", err)
	}
	defer rows.Close()

	items, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (domain.InvoiceItem, error) {
		var item domain.InvoiceItem
		err := row.Scan(
			&item.ID,
			&item.InvoiceID,
			&item.ShopID,
			&item.ShopName,
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CourierCode,
			&item.CourierService,
			&item.ShippingFee,
		)
		return item, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan invoice items failed: %w", err)
	}

	return items, nil
}

func (r *invoiceItemRepositoryImpl) SaveBulk(
	ctx context.Context,
	exec transaction.Executor,
	items []domain.InvoiceItem,
) error {
	query := `
		INSERT INTO invoice_items (
			id,
			invoice_id,
			shop_id,
			shop_name,
			product_id,
			product_name,
			quantity,
			unit_price,
			subtotal,
			courier_code,
			courier_service,
			shipping_fee_total
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id)
		DO UPDATE SET
			invoice_id = EXCLUDED.invoice_id,
			shop_id = EXCLUDED.shop_id,
			shop_name = EXCLUDED.shop_name,
			product_id = EXCLUDED.product_id,
			product_name = EXCLUDED.product_name,
			quantity = EXCLUDED.quantity,
			unit_price = EXCLUDED.unit_price,
			subtotal = EXCLUDED.subtotal,
			courier_code = EXCLUDED.courier_code,
			courier_service = EXCLUDED.courier_service,
			shipping_fee_total = EXCLUDED.shipping_fee_total
	`

	for _, item := range items {
		_, err := exec.Exec(ctx, query,
			item.ID,
			item.InvoiceID,
			item.ShopID,
			item.ShopName,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
			item.CourierCode,
			item.CourierService,
			item.ShippingFee,
		)
		if err != nil {
			return fmt.Errorf("query to save invoice item: %w", err)
		}
	}

	return nil
}
