package inventory

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type Repository interface {
	GetByProductIDAndShopID(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID) (*Inventory, error)

	ListByProductID(ctx context.Context, exec transaction.Executor, productID uuid.UUID) ([]Inventory, error)
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]Inventory, error)
	ListByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]Inventory, error)

	Create(ctx context.Context, exec transaction.Executor, inventory *Inventory) error

	Reserve(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	Release(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	Commit(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	Restock(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error

	Update(ctx context.Context, exec transaction.Executor, inventory *Inventory) error

	Delete(ctx context.Context, exec transaction.Executor, productID uuid.UUID, shopID uuid.UUID) error
}
