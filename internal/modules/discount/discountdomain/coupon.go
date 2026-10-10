package discountdomain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type CouponType string

const (
	CouponTypePercentage   CouponType = "percentage"
	CouponTypeFixedAmount  CouponType = "fixed_amount"
	CouponTypeFreeShipping CouponType = "free_shipping"
)

func (t CouponType) IsValid() bool {
	switch t {
	case CouponTypePercentage, CouponTypeFixedAmount, CouponTypeFreeShipping:
		return true
	default:
		return false
	}
}

type Coupon struct {
	ID             uuid.UUID
	Code           string
	Description    *string
	Type           CouponType
	DiscountValue  int64
	MinSpend       int64
	MaxDiscount    *int64
	QuotaTotal     int
	QuotaRemaining int
	StartsAt       time.Time
	ExpiresAt      time.Time
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      *time.Time
}

func (c *Coupon) Validate(subtotal int64, now time.Time) error {
	if !c.IsActive {
		return ErrCouponInactive
	}
	if now.Before(c.StartsAt) {
		return ErrCouponNotStarted
	}
	if !now.Before(c.ExpiresAt) {
		return ErrCouponExpired
	}
	if c.QuotaRemaining <= 0 {
		return ErrCouponQuotaExceeded
	}
	if subtotal < c.MinSpend {
		return ErrCouponMinSpendNotMet
	}
	return nil
}

func (c *Coupon) CalculateDiscount(subtotal, shippingFee int64) int64 {
	var discount int64

	switch c.Type {
	case CouponTypePercentage:
		discount = (subtotal * c.DiscountValue) / 100
		if c.MaxDiscount != nil && *c.MaxDiscount > 0 && discount > *c.MaxDiscount {
			discount = *c.MaxDiscount
		}
		if discount > subtotal {
			discount = subtotal
		}

	case CouponTypeFixedAmount:
		discount = c.DiscountValue
		if discount > subtotal {
			discount = subtotal
		}

	case CouponTypeFreeShipping:
		discount = shippingFee
		if c.MaxDiscount != nil && *c.MaxDiscount > 0 && discount > *c.MaxDiscount {
			discount = *c.MaxDiscount
		}
	}

	if discount < 0 {
		discount = 0
	}
	return discount
}

type CouponRedemption struct {
	ID             uuid.UUID
	CouponID       uuid.UUID
	CustomerID     uuid.UUID
	OrderID        uuid.UUID
	DiscountAmount int64
	RedeemedAt     time.Time
}

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
