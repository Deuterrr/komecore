package inventoryusecase

import (
	"context"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type ProductChecker interface {
	ProductExists(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (bool, error)
}

type ShopChecker interface {
	ShopExists(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) (bool, error)
}

type StockHistoryRecorder interface {
	RecordStockEvent(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID, available int) error
}
