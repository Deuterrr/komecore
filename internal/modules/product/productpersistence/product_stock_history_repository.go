package productpersistence

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
)

type ProductStockHistoryRepository struct{}

func NewProductStockHistoryRepository() *ProductStockHistoryRepository {
	return &ProductStockHistoryRepository{}
}

func (r *ProductStockHistoryRepository) RecordStockEvent(
	ctx context.Context,
	exec transaction.Executor,
	event productdomain.ProductStockEvent,
) error {
	query := `
		INSERT INTO product_stock_history (
			product_id,
			shop_id,
			available,
			recorded_at)
		VALUES ($1,$2,$3,NOW())
	`

	_, err := exec.Exec(ctx, query,
		event.ProductID,
		event.ShopID,
		event.Available,
	)
	if err != nil {
		return fmt.Errorf("record stock event failed: %w", err)
	}

	return nil
}
