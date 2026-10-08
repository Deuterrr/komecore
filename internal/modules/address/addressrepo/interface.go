package addressrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/addressdomain"

	"github.com/google/uuid"
)

type CustomerAddressRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		addressID uuid.UUID,
	) (*addressdomain.CustomerAddress, error)

	ListByIDs(
		ctx context.Context,
		exec transaction.Executor,
		addressIDs []uuid.UUID,
	) ([]addressdomain.CustomerAddress, error)

	GetDefaultByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*addressdomain.CustomerAddress, error)

	ListByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) ([]addressdomain.CustomerAddress, error)

	CountByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*int, error)

	UnsetDefaultByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		address addressdomain.CustomerAddress,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		addressID uuid.UUID,
	) error

	DeleteByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) error
}

type ShopAddressRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*addressdomain.ShopAddress, error)

	GetDefaultByShopID(
		ctx context.Context,
		exec transaction.Executor,
		shopID uuid.UUID,
	) (*addressdomain.ShopAddress, error)

	// GetDefaultsByShopIDs retrieves default address grouped by shop IDs.
	// The returned map uses the shop ID as the key and all associated
	// default address record as the value.
	//
	// Example:
	//
	//	shopIDs := []uuid.UUID{shopA, shopB}
	//
	//	result := map[uuid.UUID][]addressdomain.ShopAddress{
	//		shopA: addressDefault,
	//		shopB: addressDefault,
	//	}
	//
	// This allows callers to efficiently look up default addresses
	// belonging to a specific shop without additional filtering.
	GetDefaultsByShopIDs(
		ctx context.Context,
		exec transaction.Executor,
		shopIDs []uuid.UUID,
	) (map[uuid.UUID]addressdomain.ShopAddress, error)

	FindByShopID(
		ctx context.Context,
		exec transaction.Executor,
		shopID uuid.UUID,
	) ([]addressdomain.ShopAddress, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		address addressdomain.ShopAddress,
	) error

	Update(
		ctx context.Context,
		exec transaction.Executor,
		address addressdomain.ShopAddress,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		addressID uuid.UUID,
	) error

	UnsetActiveByShopID(
		ctx context.Context,
		exec transaction.Executor,
		shopID uuid.UUID,
	) error
}
