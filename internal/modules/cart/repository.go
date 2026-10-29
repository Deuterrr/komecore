package cart

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// Repository defines persistence operations for shopping cart aggregates, active line items,
// and draft checkout sessions within transaction boundaries.
type Repository interface {
	// Loads customer shopping cart aggregate with non-deleted line items for price calculation and basket manipulation.
	GetWithItemsByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*Cart, error)

	// Initializes and persists a new empty cart aggregate for a customer within transactional context.
	NewCart(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*Cart, error)

	// Persists cart aggregate state, synchronizing active item quantities and soft-deleting removed line items.
	Save(ctx context.Context, exec transaction.Executor, cart *Cart) error

	// Purges cart aggregate and associated line items during customer account teardown or checkout cleanup.
	DeleteByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error
}
