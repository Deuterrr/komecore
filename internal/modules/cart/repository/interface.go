package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/domain"

	"github.com/google/uuid"
)

type CartRepository interface {
	GetWithItemsByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*domain.Cart, error)

	NewCart(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) (*domain.Cart, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		cart *domain.Cart,
	) error

	DeleteByCustomerID(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
	) error
}
