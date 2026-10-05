package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/review/domain"

	"github.com/google/uuid"
)

type ListReviewsParams struct {
	ProductID uuid.UUID
	Page      int
	Limit     int
}

type ReviewRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		review *domain.Review,
	) error

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Review, error)

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	ListByProductID(
		ctx context.Context,
		exec transaction.Executor,
		params ListReviewsParams,
	) ([]domain.ReviewWithCustomer, int, error)

	GetRatingSummary(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) (*domain.ProductRatingSummary, error)

	HasReviewedOrder(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
		orderID uuid.UUID,
	) (bool, error)

	GetReviewedOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		customerID uuid.UUID,
		productID uuid.UUID,
	) ([]uuid.UUID, error)
}
