package discount

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

type ListCouponsParams struct {
	Code     *string
	IsActive *bool

	pagination.Pagination
	pagination.Sorts
}

type Repository interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Coupon, error)
	GetByCode(ctx context.Context, exec transaction.Executor, code string) (*Coupon, error)

	Save(ctx context.Context, exec transaction.Executor, coupon Coupon) error

	List(ctx context.Context, exec transaction.Executor, params ListCouponsParams) ([]Coupon, int, error)

	DecrementQuota(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	GetRedemptionByOrder(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*CouponRedemption, error)

	SaveRedemption(ctx context.Context, exec transaction.Executor, redemption CouponRedemption) error
}
