package inventoryrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"

	"github.com/google/uuid"
)

type InventoryRepository interface {
	GetByProductIDAndShopID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
	) (*inventorydomain.Inventory, error)

	ListByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) ([]inventorydomain.Inventory, error)

	// ListByProductIDs retrieves inventories grouped by product IDs.
	// The returned map uses the product ID as the key and all associated
	// inventory records as the value.
	//
	// Example:
	//
	//	productIDs := []uuid.UUID{productA, productB}
	//
	//	result := map[uuid.UUID][]inventorydomain.Inventory{
	//		productA: {
	//			inventoryA1,
	//			inventoryA2,
	//		},
	//		productB: {
	//			inventoryB1,
	//		},
	//	}
	//
	// This allows callers to efficiently look up inventories belonging to a
	// specific product without additional filtering.
	ListByProductIDs(
		ctx context.Context,
		exec transaction.Executor,
		productIDs []uuid.UUID,
	) (map[uuid.UUID][]inventorydomain.Inventory, error)

	ListByShopID(
		ctx context.Context,
		exec transaction.Executor,
		shopID uuid.UUID,
	) ([]inventorydomain.Inventory, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		inventory *inventorydomain.Inventory,
	) error

	// Reserve increments reserved_stock for
	// the given product and shop by qty
	//
	// Returns ErrInsufficientStock
	// when available stock is too low
	Reserve(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
		qty int,
	) error

	// Release decrements reserved_stock for
	// the given product and shop by qty
	Release(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
		qty int,
	) error

	// Commit decrements both stock and reserved_stock for
	// the given product and shop by qty
	Commit(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
		qty int,
	) error

	// Restock increments stock for the given product and shop by qty
	// used when a committed order is cancelled/refunded
	Restock(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
		qty int,
	) error

	Update(
		ctx context.Context,
		exec transaction.Executor,
		inventory *inventorydomain.Inventory,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		shopID uuid.UUID,
	) error
}
