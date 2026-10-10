package discountdomain

import "errors"

var (
	ErrCouponNotFound       = errors.New("coupon not found")
	ErrCouponInactive       = errors.New("coupon is inactive")
	ErrCouponExpired        = errors.New("coupon has expired")
	ErrCouponNotStarted     = errors.New("coupon is not yet active")
	ErrCouponQuotaExceeded  = errors.New("coupon quota has been exceeded")
	ErrCouponMinSpendNotMet = errors.New("minimum spend requirement not met for this coupon")
	ErrCouponAlreadyRedeemed = errors.New("coupon has already been redeemed for this order")
	ErrInvalidCouponType    = errors.New("invalid coupon type")
	ErrInvalidDiscountValue = errors.New("invalid discount value")
	ErrInvalidDates         = errors.New("expiration date must be after start date")
	ErrInvalidQuota         = errors.New("quota must be greater than zero")
	ErrDuplicateCouponCode  = errors.New("coupon code already exists")
)
