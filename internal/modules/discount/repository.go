package discount

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

// ListCouponsParams encapsulates filtering, pagination, and sorting criteria for coupon queries.
type ListCouponsParams struct {
	Code     *string
	IsActive *bool

	pagination.Pagination
	pagination.Sorts
}

// Repository defines persistence operations for promotional discounts, usage quota tracking,
// and order redemption ledger records.
type Repository interface {
	// Resolves a discount coupon record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Coupon, error)
	// Looks up an active coupon by unique alphanumeric code for checkout validation and discount application.
	GetByCode(ctx context.Context, exec transaction.Executor, code string) (*Coupon, error)

	// Persists a promotional coupon, inserting a new record or updating lifecycle quotas and active status.
	Save(ctx context.Context, exec transaction.Executor, coupon Coupon) error

	// Queries paginated coupons matching filter criteria, returning items and total matching count.
	List(ctx context.Context, exec transaction.Executor, params ListCouponsParams) ([]Coupon, int, error)

	// Atomically decrements remaining coupon quota upon order placement; prevents race conditions on limited promotions.
	DecrementQuota(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	// Resolves applied coupon redemption details associated with a specific order.
	GetRedemptionByOrder(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*CouponRedemption, error)

	// Records an immutable redemption event linking an order to a coupon and the discount value deducted.
	SaveRedemption(ctx context.Context, exec transaction.Executor, redemption CouponRedemption) error
}
