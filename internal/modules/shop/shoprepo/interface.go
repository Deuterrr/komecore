package shoprepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/shopdomain"

	"github.com/google/uuid"
)

type ShopRepository interface {
	FindByParams(
		ctx context.Context,
		exec transaction.Executor,
		params FindShopsParams,
	) ([]shopdomain.Shop, int, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*shopdomain.Shop, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		slug string,
	) (*shopdomain.Shop, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]shopdomain.Shop, error)
	// GetActive() ([]shopdomain.Shop, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		shop shopdomain.Shop,
	) error
	// Update(shop shopdomain.Shop) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	// GetSupportedCouriers(shopID uuid.UUID) ([]Courier, error)
}
