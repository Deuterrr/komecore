package address

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type CustomerAddressRepository interface {
	GetByID(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) (*CustomerAddress, error)
	GetDefaultByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*CustomerAddress, error)

	ListByIDs(ctx context.Context, exec transaction.Executor, addressIDs []uuid.UUID) ([]CustomerAddress, error)
	ListByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) ([]CustomerAddress, error)

	CountByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*int, error)

	UnsetDefaultByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error

	Save(ctx context.Context, exec transaction.Executor, address CustomerAddress) error

	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error
	DeleteByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) error
}

type ShopAddressRepository interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*ShopAddress, error)
	GetDefaultByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) (*ShopAddress, error)
	GetDefaultsByShopIDs(ctx context.Context, exec transaction.Executor, shopIDs []uuid.UUID) (map[uuid.UUID]ShopAddress, error)

	FindByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]ShopAddress, error)

	Create(ctx context.Context, exec transaction.Executor, address ShopAddress) error

	Update(ctx context.Context, exec transaction.Executor, address ShopAddress) error

	Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error

	UnsetActiveByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) error
}
