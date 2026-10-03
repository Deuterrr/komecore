package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/domain"

	"github.com/google/uuid"
)

type ShopRepository interface {
	FindByParams(
		ctx context.Context,
		exec transaction.Executor,
		params FindShopsParams,
	) ([]domain.Shop, int, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Shop, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		slug string,
	) (*domain.Shop, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]domain.Shop, error)
	// GetActive() ([]domain.Shop, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		shop domain.Shop,
	) error
	// Update(shop domain.Shop) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	// GetSupportedCouriers(shopID uuid.UUID) ([]Courier, error)
}
