package discountrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/discount/discountdomain"

	"github.com/google/uuid"
)

type CouponRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*discountdomain.Coupon, error)

	GetByCode(
		ctx context.Context,
		exec transaction.Executor,
		code string,
	) (*discountdomain.Coupon, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		coupon discountdomain.Coupon,
	) error

	List(
		ctx context.Context,
		exec transaction.Executor,
		params ListCouponsParams,
	) ([]discountdomain.Coupon, int, error)

	DecrementQuota(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	SaveRedemption(
		ctx context.Context,
		exec transaction.Executor,
		redemption discountdomain.CouponRedemption,
	) error

	GetRedemptionByOrder(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*discountdomain.CouponRedemption, error)
}
