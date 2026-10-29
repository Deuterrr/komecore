package address

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// CustomerAddressRepository defines persistence operations for customer shipping destinations,
// coordinate geocoding, and default address lifecycle invariants within transaction boundaries.
type CustomerAddressRepository interface {
	// Resolves a customer address by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) (*CustomerAddress, error)
	// Resolves customer's active default shipping destination used for checkout fulfillment estimation.
	GetDefaultByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*CustomerAddress, error)

	// Batch-loads addresses by identifiers to eliminate N+1 roundtrips during checkout pricing.
	ListByIDs(ctx context.Context, exec transaction.Executor, addressIDs []uuid.UUID) ([]CustomerAddress, error)
	// Queries all active delivery destinations registered by a customer.
	ListByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) ([]CustomerAddress, error)

	// Counts customer addresses to enforce the per-customer address capacity limit prior to insertion.
	CountByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*int, error)

	// Clears default status across customer addresses before designating a new primary shipping destination.
	UnsetDefaultByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error

	// Inserts or updates an address; auto-promotes the first address created for a customer to default.
	Save(ctx context.Context, exec transaction.Executor, address CustomerAddress) error

	// Removes an address record; invariant guards reject deletion if the target address is the default.
	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error
	// Cascades removal of all registered addresses when customer profile or account is purged.
	DeleteByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error
}

// ShopAddressRepository defines persistence operations for merchant physical branches,
// warehouse pickup origins, and dispatch locations.
type ShopAddressRepository interface {
	// Resolves a merchant branch address by primary identifier.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*ShopAddress, error)
	// Resolves active primary dispatch origin used as fulfillment source for courier rate calculation.
	GetDefaultByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) (*ShopAddress, error)
	// Batch-resolves default origin addresses keyed by shop ID to eliminate N+1 queries during multi-shop checkout.
	GetDefaultsByShopIDs(ctx context.Context, exec transaction.Executor, shopIDs []uuid.UUID) (map[uuid.UUID]ShopAddress, error)

	// Lists all active and inactive branch addresses registered under a merchant shop.
	FindByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]ShopAddress, error)

	// Persists a new merchant origin address record within transactional boundaries.
	Create(ctx context.Context, exec transaction.Executor, address ShopAddress) error

	// Updates branch address attributes, contact details, or geocoded coordinates.
	Update(ctx context.Context, exec transaction.Executor, address ShopAddress) error

	// Removes a merchant branch address record from persistence.
	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error

	// Clears active flag across existing shop addresses before activating a new primary dispatch origin.
	UnsetActiveByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) error
}
