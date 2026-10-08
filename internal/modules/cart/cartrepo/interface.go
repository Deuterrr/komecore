package cartrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/cartdomain"

	"github.com/google/uuid"
)

type CartRepository interface {
	GetWithItemsByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*cartdomain.Cart, error)

	NewCart(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*cartdomain.Cart, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		cart *cartdomain.Cart,
	) error

	DeleteByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) error
}
