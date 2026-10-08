package wishlistrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/wishlist/wishlistdomain"

	"github.com/google/uuid"
)

type WishlistRepository interface {
	Add(
		ctx context.Context,
		exec transaction.Executor,
		item wishlistdomain.WishlistItem,
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
	) ([]wishlistdomain.WishlistItem, error)

	Exists(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
	) (bool, error)
}
