package wishlist

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type Repository interface {
	Add(
		ctx context.Context,
		exec transaction.Executor,
		item WishlistItem,
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
	) ([]WishlistItem, error)

	Exists(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
	) (bool, error)
}

type WishlistRepository = Repository
