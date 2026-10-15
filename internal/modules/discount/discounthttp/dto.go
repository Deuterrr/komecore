package discounthttp

import (
	"time"

	"komecore/internal/httpx"
	"komecore/internal/modules/discount"

	"github.com/google/uuid"
)

type ValidateCouponRequest struct {
	Code        string `json:"code"`
	Subtotal    int64  `json:"subtotal"`
	ShippingFee int64  `json:"shipping_fee"`
}

type ValidateCouponResponse struct {
	Code           string              `json:"code"`
	Type           discount.CouponType `json:"type"`
	DiscountValue  int64               `json:"discount_value"`
	DiscountAmount int64               `json:"discount_amount"`
	Subtotal       int64               `json:"subtotal"`
	ShippingFee    int64               `json:"shipping_fee"`
	FinalTotal     int64               `json:"final_total"`
}

type CreateCouponRequest struct {
	Code          string              `json:"code"`
	Description   *string             `json:"description,omitempty"`
	Type          discount.CouponType `json:"type"`
	DiscountValue int64               `json:"discount_value"`
	MinSpend      int64               `json:"min_spend"`
	MaxDiscount   *int64              `json:"max_discount,omitempty"`
	QuotaTotal    int                 `json:"quota_total"`
	StartsAt      time.Time           `json:"starts_at"`
	ExpiresAt     time.Time           `json:"expires_at"`
	IsActive      bool                `json:"is_active"`
}

type CouponResponse struct {
	ID             uuid.UUID           `json:"id"`
	Code           string              `json:"code"`
	Description    *string             `json:"description"`
	Type           discount.CouponType `json:"type"`
	DiscountValue  int64               `json:"discount_value"`
	MinSpend       int64               `json:"min_spend"`
	MaxDiscount    *int64              `json:"max_discount"`
	QuotaTotal     int                 `json:"quota_total"`
	QuotaRemaining int                 `json:"quota_remaining"`
	StartsAt       time.Time           `json:"starts_at"`
	ExpiresAt      time.Time           `json:"expires_at"`
	IsActive       bool                `json:"is_active"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      *time.Time          `json:"updated_at"`
}

type ListCouponsResponse = httpx.PaginatedResponse[CouponResponse]
