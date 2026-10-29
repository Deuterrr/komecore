package discount

import (
	"context"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type Service struct {
	couponRepo Repository
	executor   transaction.Executor
	clock      appclock.Clock
}

func NewService(
	couponRepo Repository,
	executor transaction.Executor,
	clock appclock.Clock,
) *Service {
	if clock == nil {
		clock = appclock.RealClock{}
	}
	return &Service{
		couponRepo: couponRepo,
		executor:   executor,
		clock:      clock,
	}
}

type ValidateCouponInput struct {
	Code        string
	Subtotal    int64
	ShippingFee int64
	Now         *time.Time
}

type ValidateCouponResult struct {
	Coupon         Coupon
	DiscountAmount int64
	Subtotal       int64
	ShippingFee    int64
	FinalTotal     int64
}

func (s *Service) ValidateCoupon(ctx context.Context, input ValidateCouponInput) (*ValidateCouponResult, error) {
	code := NormalizeCode(input.Code)
	if code == "" {
		return nil, ErrCouponNotFound
	}

	coupon, err := s.couponRepo.GetByCode(ctx, s.executor, code)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup coupon: %w", err)
	}
	if coupon == nil {
		return nil, ErrCouponNotFound
	}

	now := s.clock.Now()
	if input.Now != nil {
		now = *input.Now
	}

	if err := coupon.Validate(input.Subtotal, now); err != nil {
		return nil, err
	}

	discountAmount := coupon.CalculateDiscount(input.Subtotal, input.ShippingFee)
	finalTotal := input.Subtotal + input.ShippingFee - discountAmount
	if finalTotal < 0 {
		finalTotal = 0
	}

	return &ValidateCouponResult{
		Coupon:         *coupon,
		DiscountAmount: discountAmount,
		Subtotal:       input.Subtotal,
		ShippingFee:    input.ShippingFee,
		FinalTotal:     finalTotal,
	}, nil
}

type RedeemCouponInput struct {
	Code        string
	CustomerID  uuid.UUID
	OrderID     uuid.UUID
	Subtotal    int64
	ShippingFee int64
}

func (s *Service) RedeemCoupon(ctx context.Context, exec transaction.Executor, input RedeemCouponInput) (*CouponRedemption, error) {
	code := NormalizeCode(input.Code)
	if code == "" {
		return nil, ErrCouponNotFound
	}

	if exec == nil {
		exec = s.executor
	}

	coupon, err := s.couponRepo.GetByCode(ctx, exec, code)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup coupon: %w", err)
	}
	if coupon == nil {
		return nil, ErrCouponNotFound
	}

	if err := coupon.Validate(input.Subtotal, s.clock.Now()); err != nil {
		return nil, err
	}

	// Check if already redeemed for this order
	existing, err := s.couponRepo.GetRedemptionByOrder(ctx, exec, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing redemption: %w", err)
	}
	if existing != nil {
		return nil, ErrCouponAlreadyRedeemed
	}

	if err := s.couponRepo.DecrementQuota(ctx, exec, coupon.ID); err != nil {
		return nil, err
	}

	discountAmount := coupon.CalculateDiscount(input.Subtotal, input.ShippingFee)

	redemption := CouponRedemption{
		ID:             uuid.New(),
		CouponID:       coupon.ID,
		CustomerID:     input.CustomerID,
		OrderID:        input.OrderID,
		DiscountAmount: discountAmount,
		RedeemedAt:     s.clock.Now(),
	}

	if err := s.couponRepo.SaveRedemption(ctx, exec, redemption); err != nil {
		return nil, fmt.Errorf("failed to record redemption: %w", err)
	}

	return &redemption, nil
}

type CreateCouponInput struct {
	Code          string
	Description   *string
	Type          CouponType
	DiscountValue int64
	MinSpend      int64
	MaxDiscount   *int64
	QuotaTotal    int
	StartsAt      time.Time
	ExpiresAt     time.Time
	IsActive      bool
}

func (s *Service) CreateCoupon(ctx context.Context, input CreateCouponInput) (*Coupon, error) {
	code := NormalizeCode(input.Code)
	if code == "" {
		return nil, fmt.Errorf("coupon code cannot be empty")
	}

	if !input.Type.IsValid() {
		return nil, ErrInvalidCouponType
	}
	if input.DiscountValue <= 0 {
		return nil, ErrInvalidDiscountValue
	}
	if input.Type == CouponTypePercentage && input.DiscountValue > 100 {
		return nil, fmt.Errorf("percentage discount cannot exceed 100%%: %w", ErrInvalidDiscountValue)
	}
	if input.QuotaTotal <= 0 {
		return nil, ErrInvalidQuota
	}
	if !input.ExpiresAt.After(input.StartsAt) {
		return nil, ErrInvalidDates
	}

	existing, err := s.couponRepo.GetByCode(ctx, s.executor, code)
	if err != nil {
		return nil, fmt.Errorf("failed to verify code uniqueness: %w", err)
	}
	if existing != nil {
		return nil, ErrDuplicateCouponCode
	}

	now := s.clock.Now()
	coupon := Coupon{
		ID:             uuid.New(),
		Code:           code,
		Description:    input.Description,
		Type:           input.Type,
		DiscountValue:  input.DiscountValue,
		MinSpend:       input.MinSpend,
		MaxDiscount:    input.MaxDiscount,
		QuotaTotal:     input.QuotaTotal,
		QuotaRemaining: input.QuotaTotal,
		StartsAt:       input.StartsAt,
		ExpiresAt:      input.ExpiresAt,
		IsActive:       input.IsActive,
		CreatedAt:      now,
	}

	if err := s.couponRepo.Save(ctx, s.executor, coupon); err != nil {
		return nil, fmt.Errorf("failed to save coupon: %w", err)
	}

	return &coupon, nil
}

type ListCouponsInput struct {
	Code     *string
	IsActive *bool
	Page     int
	Limit    int
}

func (s *Service) ListCoupons(ctx context.Context, input ListCouponsInput) ([]Coupon, int, error) {
	params := ListCouponsParams{
		Code:     input.Code,
		IsActive: input.IsActive,
	}
	params.Pagination.Page = input.Page
	params.Pagination.Limit = input.Limit

	return s.couponRepo.List(ctx, s.executor, params)
}
