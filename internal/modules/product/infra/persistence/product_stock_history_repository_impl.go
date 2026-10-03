package persistence

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
)

type productStockHistoryRepositoryImpl struct{}

func NewProductStockHistoryRepository() repository.ProductStockHistoryRepository {
	return &productStockHistoryRepositoryImpl{}
}

func (r *productStockHistoryRepositoryImpl) RecordStockEvent(
	ctx context.Context,
	exec transaction.Executor,
	event domain.ProductStockEvent,
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
