package inventory

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// Repository defines persistence operations for multi-shop warehouse stock allocations,
// two-phase stock holds (reserve/commit/release), and physical inventory reconciliation.
type Repository interface {
	// Resolves inventory balance and reserved quantity for a specific product within a merchant shop.
	GetByProductIDAndShopID(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID) (*Inventory, error)

	// Queries stock balances across all merchant shops holding inventory for a single catalog product.
	ListByProductID(ctx context.Context, exec transaction.Executor, productID uuid.UUID) ([]Inventory, error)
	// Batch-loads inventory rows grouped by product ID to prevent N+1 queries during multi-shop basket evaluation.
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]Inventory, error)
	// Queries all product stock allocations maintained under a merchant shop.
	ListByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]Inventory, error)

	// Initializes a new product inventory allocation row with initial total stock.
	Create(ctx context.Context, exec transaction.Executor, inventory *Inventory) error

	// Places a temporary hold by incrementing reserved_stock; ensures available balance (total - reserved) is sufficient.
	Reserve(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	// Unlocks held stock by decrementing reserved_stock following payment timeout or customer order cancellation.
	Release(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	// Finalizes stock deduction by decrementing both total_stock and reserved_stock upon order confirmation or fulfillment.
	Commit(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	// Increments total_stock when cancelled or refunded orders return items back to merchant warehouse stock.
	Restock(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	// Mutates base inventory configuration and safety stock thresholds.
	Update(ctx context.Context, exec transaction.Executor, inventory *Inventory) error

	// Removes an inventory record when a product is permanently delisted from a merchant shop catalog.
	Delete(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID) error
}
