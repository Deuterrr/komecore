package review

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type ListReviewsParams struct {
	ProductID uuid.UUID
	Page      int
	Limit     int
}

type Repository interface {
	Create(ctx context.Context, exec transaction.Executor, review *Review) error

	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Review, error)

	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	ListByProductID(ctx context.Context, exec transaction.Executor, params ListReviewsParams) ([]ReviewWithCustomer, int, error)

	GetRatingSummary(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (*ProductRatingSummary, error)

	HasReviewedOrder(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID, orderID uuid.UUID) (bool, error)

	GetReviewedOrderIDs(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID) ([]uuid.UUID, error)
}
