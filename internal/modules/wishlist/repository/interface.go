package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/wishlist/domain"

	"github.com/google/uuid"
)

type WishlistRepository interface {
	Add(
		ctx context.Context,
		exec transaction.Executor,
		item domain.WishlistItem,
	) error

	Remove(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
	) error

	ListByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) ([]domain.WishlistItem, error)

	Exists(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
	) (bool, error)
}
