package cart

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type Repository interface {
	GetWithItemsByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*Cart, error)

	NewCart(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*Cart, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		cart *Cart,
	) error

	DeleteByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) error
}

type CartRepository = Repository
