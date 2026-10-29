package wishlist

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// Repository defines persistence operations for customer wishlists, bookmarked products,
// and saved favorite collections.
type Repository interface {
	// Saves a product item to customer wishlist; idempotent against duplicate additions.
	Add(ctx context.Context, exec transaction.Executor, item WishlistItem) error

	// Removes a bookmarked product from customer wishlist.
	Remove(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID) error

	// Queries all active wishlist items bookmarked by a customer.
	ListByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) ([]WishlistItem, error)

	// Checks whether a product already exists in customer wishlist to prevent redundant writes.
	Exists(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID) (bool, error)
}
