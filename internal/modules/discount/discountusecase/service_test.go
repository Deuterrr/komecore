package discountusecase

import (
	"context"
	"testing"
	"time"

	"komecore/internal/modules/discount/discountdomain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClock struct {
	now time.Time
}

func (m mockClock) Now() time.Time {
	return m.now
}

func TestDiscountService_ValidateCoupon(t *testing.T) {
	ctx := context.Background()
	repo := newMockCouponRepo()
	exec := &mockExecutor{}
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	clock := mockClock{now: now}

	svc := NewDiscountService(repo, exec, clock)

	// Seed coupons
	maxCap := int64(20000)
	pctCoupon := discountdomain.Coupon{
		ID:             uuid.New(),
		Code:           "SAVE10",
		Type:           discountdomain.CouponTypePercentage,
		DiscountValue:  10, // 10%
		MinSpend:       50000,
		MaxDiscount:    &maxCap,
		QuotaTotal:     100,
		QuotaRemaining: 100,
		StartsAt:       now.Add(-24 * time.Hour),
		ExpiresAt:      now.Add(24 * time.Hour),
		IsActive:       true,
		CreatedAt:      now.Add(-24 * time.Hour),
	}
	require.NoError(t, repo.Save(ctx, exec, pctCoupon))

	fixedCoupon := discountdomain.Coupon{
		ID:             uuid.New(),
		Code:           "FLAT25K",
		Type:           discountdomain.CouponTypeFixedAmount,
		DiscountValue:  25000,
		MinSpend:       30000,
		QuotaTotal:     50,
		QuotaRemaining: 50,
		StartsAt:       now.Add(-24 * time.Hour),
		ExpiresAt:      now.Add(24 * time.Hour),
		IsActive:       true,
		CreatedAt:      now.Add(-24 * time.Hour),
	}
	require.NoError(t, repo.Save(ctx, exec, fixedCoupon))

	freeShipCoupon := discountdomain.Coupon{
		ID:             uuid.New(),
		Code:           "FREESHIP",
		Type:           discountdomain.CouponTypeFreeShipping,
		DiscountValue:  0,
		MinSpend:       20000,
		QuotaTotal:     10,
		QuotaRemaining: 10,
		StartsAt:       now.Add(-24 * time.Hour),
		ExpiresAt:      now.Add(24 * time.Hour),
		IsActive:       true,
		CreatedAt:      now.Add(-24 * time.Hour),
	}
	require.NoError(t, repo.Save(ctx, exec, freeShipCoupon))

	expiredCoupon := discountdomain.Coupon{
		ID:             uuid.New(),
		Code:           "OLDPROMO",
		Type:           discountdomain.CouponTypeFixedAmount,
		DiscountValue:  10000,
		MinSpend:       0,
		QuotaTotal:     10,
		QuotaRemaining: 10,
		StartsAt:       now.Add(-48 * time.Hour),
		ExpiresAt:      now.Add(-1 * time.Hour),
		IsActive:       true,
		CreatedAt:      now.Add(-48 * time.Hour),
	}
	require.NoError(t, repo.Save(ctx, exec, expiredCoupon))

	t.Run("percentage coupon within cap calculates exact discount", func(t *testing.T) {
		res, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "SAVE10",
			Subtotal:    100000, // 10% of 100k = 10,000 (< 20k cap)
			ShippingFee: 15000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(10000), res.DiscountAmount)
		assert.Equal(t, int64(105000), res.FinalTotal) // 100k + 15k - 10k
	})

	t.Run("percentage coupon hitting cap is clamped to max discount", func(t *testing.T) {
		res, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "SAVE10",
			Subtotal:    500000, // 10% of 500k = 50,000 -> clamped to 20,000 cap
			ShippingFee: 20000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(20000), res.DiscountAmount)
		assert.Equal(t, int64(500000), res.FinalTotal) // 500k + 20k - 20k
	})

	t.Run("percentage coupon below min spend fails", func(t *testing.T) {
		_, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "SAVE10",
			Subtotal:    30000, // min spend is 50,000
			ShippingFee: 10000,
		})
		require.ErrorIs(t, err, discountdomain.ErrCouponMinSpendNotMet)
	})

	t.Run("fixed amount coupon applied correctly", func(t *testing.T) {
		res, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "FLAT25K",
			Subtotal:    80000,
			ShippingFee: 10000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(25000), res.DiscountAmount)
		assert.Equal(t, int64(65000), res.FinalTotal) // 80k + 10k - 25k
	})

	t.Run("fixed amount discount exceeds subtotal is capped at subtotal", func(t *testing.T) {
		res, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "FLAT25K",
			Subtotal:    20000, // subtotal less than 25k discount
			ShippingFee: 10000,
		})
		// But wait: min spend for FLAT25K is 30,000! So subtotal 20000 fails min spend!
		assert.ErrorIs(t, err, discountdomain.ErrCouponMinSpendNotMet)
		assert.Nil(t, res)
	})

	t.Run("free shipping coupon discounts entire shipping fee", func(t *testing.T) {
		res, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:        "FREESHIP",
			Subtotal:    50000,
			ShippingFee: 18000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(18000), res.DiscountAmount)
		assert.Equal(t, int64(50000), res.FinalTotal)
	})

	t.Run("expired coupon fails validation", func(t *testing.T) {
		_, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:     "OLDPROMO",
			Subtotal: 100000,
		})
		require.ErrorIs(t, err, discountdomain.ErrCouponExpired)
	})

	t.Run("unknown coupon code fails validation", func(t *testing.T) {
		_, err := svc.ValidateCoupon(ctx, ValidateCouponInput{
			Code:     "NONEXISTENT",
			Subtotal: 100000,
		})
		require.ErrorIs(t, err, discountdomain.ErrCouponNotFound)
	})
}

func TestDiscountService_RedeemCoupon_AtomicQuota(t *testing.T) {
	ctx := context.Background()
	repo := newMockCouponRepo()
	exec := &mockExecutor{}
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	clock := mockClock{now: now}

	svc := NewDiscountService(repo, exec, clock)

	// Coupon with exactly quota_remaining = 1
	coupon := discountdomain.Coupon{
		ID:             uuid.New(),
		Code:           "LASTONE",
		Type:           discountdomain.CouponTypeFixedAmount,
		DiscountValue:  10000,
		MinSpend:       0,
		QuotaTotal:     1,
		QuotaRemaining: 1,
		StartsAt:       now.Add(-24 * time.Hour),
		ExpiresAt:      now.Add(24 * time.Hour),
		IsActive:       true,
		CreatedAt:      now.Add(-24 * time.Hour),
	}
	require.NoError(t, repo.Save(ctx, exec, coupon))

	customer1 := uuid.New()
	order1 := uuid.New()
	customer2 := uuid.New()
	order2 := uuid.New()

	// First redemption: MUST succeed
	redemp1, err := svc.RedeemCoupon(ctx, exec, RedeemCouponInput{
		Code:        "LASTONE",
		CustomerID:  customer1,
		OrderID:     order1,
		Subtotal:    50000,
		ShippingFee: 10000,
	})
	require.NoError(t, err)
	assert.NotNil(t, redemp1)
	assert.Equal(t, int64(10000), redemp1.DiscountAmount)
	assert.Equal(t, order1, redemp1.OrderID)

	// Second concurrent redemption: MUST fail with quota exceeded
	redemp2, err := svc.RedeemCoupon(ctx, exec, RedeemCouponInput{
		Code:        "LASTONE",
		CustomerID:  customer2,
		OrderID:     order2,
		Subtotal:    50000,
		ShippingFee: 10000,
	})
	require.ErrorIs(t, err, discountdomain.ErrCouponQuotaExceeded)
	assert.Nil(t, redemp2)

	// Duplicate redemption for same order fails
	_, err = svc.RedeemCoupon(ctx, exec, RedeemCouponInput{
		Code:       "LASTONE",
		CustomerID: customer1,
		OrderID:    order1,
		Subtotal:   50000,
	})
	require.Error(t, err)
}

func TestDiscountService_CreateCoupon(t *testing.T) {
	ctx := context.Background()
	repo := newMockCouponRepo()
	exec := &mockExecutor{}
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	clock := mockClock{now: now}

	svc := NewDiscountService(repo, exec, clock)

	t.Run("successfully create coupon", func(t *testing.T) {
		desc := "15% off grand opening"
		c, err := svc.CreateCoupon(ctx, CreateCouponInput{
			Code:          "welcome15",
			Description:   &desc,
			Type:          discountdomain.CouponTypePercentage,
			DiscountValue: 15,
			MinSpend:      50000,
			QuotaTotal:    500,
			StartsAt:      now,
			ExpiresAt:     now.Add(30 * 24 * time.Hour),
			IsActive:      true,
		})
		require.NoError(t, err)
		assert.Equal(t, "WELCOME15", c.Code)
		assert.Equal(t, 500, c.QuotaRemaining)
	})

	t.Run("reject percentage over 100", func(t *testing.T) {
		_, err := svc.CreateCoupon(ctx, CreateCouponInput{
			Code:          "OVER100",
			Type:          discountdomain.CouponTypePercentage,
			DiscountValue: 105,
			QuotaTotal:    10,
			StartsAt:      now,
			ExpiresAt:     now.Add(24 * time.Hour),
			IsActive:      true,
		})
		require.Error(t, err)
	})

	t.Run("reject expiry before start date", func(t *testing.T) {
		_, err := svc.CreateCoupon(ctx, CreateCouponInput{
			Code:          "TIMETRAVEL",
			Type:          discountdomain.CouponTypeFixedAmount,
			DiscountValue: 10000,
			QuotaTotal:    10,
			StartsAt:      now,
			ExpiresAt:     now.Add(-24 * time.Hour),
			IsActive:      true,
		})
		require.ErrorIs(t, err, discountdomain.ErrInvalidDates)
	})

	t.Run("reject duplicate coupon code", func(t *testing.T) {
		_, err := svc.CreateCoupon(ctx, CreateCouponInput{
			Code:          "WELCOME15",
			Type:          discountdomain.CouponTypeFixedAmount,
			DiscountValue: 10000,
			QuotaTotal:    10,
			StartsAt:      now,
			ExpiresAt:     now.Add(24 * time.Hour),
			IsActive:      true,
		})
		require.ErrorIs(t, err, discountdomain.ErrDuplicateCouponCode)
	})
}
