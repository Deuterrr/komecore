package review

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// ListReviewsParams encapsulates pagination parameters for product customer review listings.
type ListReviewsParams struct {
	ProductID uuid.UUID
	Page      int
	Limit     int
}

// Repository defines persistence operations for verified customer reviews, star ratings,
// and product feedback summaries within transaction boundaries.
type Repository interface {
	// Persists a customer review, recording star rating and feedback comment for a purchased product.
	Create(ctx context.Context, exec transaction.Executor, review *Review) error

	// Resolves a review record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Review, error)

	// Soft-deletes or removes a review record from public display.
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	// Queries paginated customer reviews joined with reviewer profile information for storefront display.
	ListByProductID(ctx context.Context, exec transaction.Executor, params ListReviewsParams) ([]ReviewWithCustomer, int, error)

	// Aggregates star rating distribution, average score, and total review count for a product.
	GetRatingSummary(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (*ProductRatingSummary, error)

	// Verifies whether a customer has already submitted a review for a specific product in an order to prevent duplicates.
	HasReviewedOrder(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID, orderID uuid.UUID) (bool, error)

	// Queries all order IDs where the customer previously reviewed the product to enforce single-review-per-order invariant.
	GetReviewedOrderIDs(ctx context.Context, exec transaction.Executor, customerID uuid.UUID, productID uuid.UUID) ([]uuid.UUID, error)
}
