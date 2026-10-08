package shipmentusecase

import (
	"context"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type OrderDeliveryUpdater interface {
	MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error
}
