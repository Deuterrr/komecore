package discountusecase

import (
	"context"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/discount"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type CouponRepository interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*discount.Coupon, error)
	GetByCode(ctx context.Context, exec transaction.Executor, code string) (*discount.Coupon, error)
	Save(ctx context.Context, exec transaction.Executor, coupon discount.Coupon) error
	List(ctx context.Context, exec transaction.Executor, params discount.ListCouponsParams) ([]discount.Coupon, int, error)
	DecrementQuota(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	SaveRedemption(ctx context.Context, exec transaction.Executor, redemption discount.CouponRedemption) error
	GetRedemptionByOrder(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*discount.CouponRedemption, error)
}

type ValidateCouponInput struct {
	Code        string
	Subtotal    int64
	ShippingFee int64
	Now         *time.Time
}

type ValidateCouponResult struct {
	Coupon         discount.Coupon
	DiscountAmount int64
	Subtotal       int64
	ShippingFee    int64
	FinalTotal     int64
}

type RedeemCouponInput struct {
	Code        string
	CustomerID  uuid.UUID
	OrderID     uuid.UUID
	Subtotal    int64
	ShippingFee int64
}

type CreateCouponInput struct {
	Code          string
	Description   *string
	Type          discount.CouponType
	DiscountValue int64
	MinSpend      int64
	MaxDiscount   *int64
	QuotaTotal    int
	StartsAt      time.Time
	ExpiresAt     time.Time
	IsActive      bool
}

type ListCouponsInput struct {
	Code     *string
	IsActive *bool
	Page     int
	Limit    int
}

type DiscountService struct {
	couponRepo CouponRepository
	executor   transaction.Executor
	clock      appclock.Clock
}

func NewDiscountService(
	couponRepo CouponRepository,
	executor transaction.Executor,
	clock appclock.Clock,
) *DiscountService {
	if clock == nil {
		clock = appclock.RealClock{}
	}
	return &DiscountService{
		couponRepo: couponRepo,
		executor:   executor,
		clock:      clock,
	}
}

func (s *DiscountService) ValidateCoupon(
	ctx context.Context,
	input ValidateCouponInput,
) (*ValidateCouponResult, error) {
	code := discount.NormalizeCode(input.Code)
	if code == "" {
		return nil, discount.ErrCouponNotFound
	}

	coupon, err := s.couponRepo.GetByCode(ctx, s.executor, code)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup coupon: %w", err)
	}
	if coupon == nil {
		return nil, discount.ErrCouponNotFound
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

func (s *DiscountService) RedeemCoupon(
	ctx context.Context,
	exec transaction.Executor,
	input RedeemCouponInput,
) (*discount.CouponRedemption, error) {
	code := discount.NormalizeCode(input.Code)
	if code == "" {
		return nil, discount.ErrCouponNotFound
	}

	if exec == nil {
		exec = s.executor
	}

	coupon, err := s.couponRepo.GetByCode(ctx, exec, code)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup coupon: %w", err)
	}
	if coupon == nil {
		return nil, discount.ErrCouponNotFound
	}

	// Validate status, date, quota, min spend
	if err := coupon.Validate(input.Subtotal, s.clock.Now()); err != nil {
		return nil, err
	}

	// Check if already redeemed for this order
	existing, err := s.couponRepo.GetRedemptionByOrder(ctx, exec, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing redemption: %w", err)
	}
	if existing != nil {
		return nil, discount.ErrCouponAlreadyRedeemed
	}

	// Atomically decrement quota
	if err := s.couponRepo.DecrementQuota(ctx, exec, coupon.ID); err != nil {
		return nil, err
	}

	discountAmount := coupon.CalculateDiscount(input.Subtotal, input.ShippingFee)

	redemption := discount.CouponRedemption{
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

func (s *DiscountService) CreateCoupon(
	ctx context.Context,
	input CreateCouponInput,
) (*discount.Coupon, error) {
	code := discount.NormalizeCode(input.Code)
	if code == "" {
		return nil, fmt.Errorf("coupon code cannot be empty")
	}

	if !input.Type.IsValid() {
		return nil, discount.ErrInvalidCouponType
	}

	if input.DiscountValue <= 0 {
		return nil, discount.ErrInvalidDiscountValue
	}

	if input.Type == discount.CouponTypePercentage && input.DiscountValue > 100 {
		return nil, fmt.Errorf("percentage discount cannot exceed 100%%: %w", discount.ErrInvalidDiscountValue)
	}

	if input.QuotaTotal <= 0 {
		return nil, discount.ErrInvalidQuota
	}

	if !input.ExpiresAt.After(input.StartsAt) {
		return nil, discount.ErrInvalidDates
	}

	existing, err := s.couponRepo.GetByCode(ctx, s.executor, code)
	if err != nil {
		return nil, fmt.Errorf("failed to verify code uniqueness: %w", err)
	}
	if existing != nil {
		return nil, discount.ErrDuplicateCouponCode
	}

	now := s.clock.Now()
	coupon := discount.Coupon{
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

func (s *DiscountService) ListCoupons(
	ctx context.Context,
	input ListCouponsInput,
) ([]discount.Coupon, int, error) {
	params := discount.ListCouponsParams{
		Code:     input.Code,
		IsActive: input.IsActive,
	}
	params.Pagination.Page = input.Page
	params.Pagination.Limit = input.Limit

	return s.couponRepo.List(ctx, s.executor, params)
}
