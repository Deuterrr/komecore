package discounthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/discount/discountdomain"
	"komecore/internal/modules/discount/discounthttp"
	"komecore/internal/modules/discount/discountrepo"
	"komecore/internal/modules/discount/discountusecase"
	"komecore/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	coupon     *discountdomain.Coupon
	saveErr    error
	listResult []discountdomain.Coupon
	listTotal  int
}

func (m *mockRepo) GetByCode(_ context.Context, _ transaction.Executor, code string) (*discountdomain.Coupon, error) {
	if m.coupon != nil && m.coupon.Code == code {
		return m.coupon, nil
	}
	return nil, nil
}

func (m *mockRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*discountdomain.Coupon, error) {
	if m.coupon != nil && m.coupon.ID == id {
		return m.coupon, nil
	}
	return nil, nil
}

func (m *mockRepo) Save(_ context.Context, _ transaction.Executor, c discountdomain.Coupon) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.coupon = &c
	return nil
}

func (m *mockRepo) List(_ context.Context, _ transaction.Executor, _ discountrepo.ListCouponsParams) ([]discountdomain.Coupon, int, error) {
	return m.listResult, m.listTotal, nil
}

func (m *mockRepo) DecrementQuota(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) SaveRedemption(_ context.Context, _ transaction.Executor, _ discountdomain.CouponRedemption) error {
	return nil
}

func (m *mockRepo) GetRedemptionByOrder(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*discountdomain.CouponRedemption, error) {
	return nil, nil
}

func setupHandler(repo *mockRepo, now time.Time) *discounthttp.DiscountHandler {
	svc := discountusecase.NewDiscountService(repo, nil, clock.NewMockClock(now))
	return discounthttp.NewDiscountHandler(svc)
}

func TestDiscountHandler_ValidateCoupon_Success(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		coupon: &discountdomain.Coupon{
			ID:             uuid.New(),
			Code:           "SAVE10",
			Type:           discountdomain.CouponTypePercentage,
			DiscountValue:  10,
			MinSpend:       100000,
			QuotaTotal:     10,
			QuotaRemaining: 10,
			StartsAt:       now.Add(-time.Hour),
			ExpiresAt:      now.Add(time.Hour),
			IsActive:       true,
		},
	}
	h := setupHandler(repo, now)

	body, _ := json.Marshal(discounthttp.ValidateCouponRequest{
		Code:        "SAVE10",
		Subtotal:    200000,
		ShippingFee: 15000,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := h.ValidateCoupon(w, req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp discounthttp.ValidateCouponResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "SAVE10", resp.Code)
	assert.Equal(t, int64(20000), resp.DiscountAmount)
	assert.Equal(t, int64(195000), resp.FinalTotal)
}

func TestDiscountHandler_ValidateCoupon_BadRequest(t *testing.T) {
	h := setupHandler(&mockRepo{}, time.Now())

	// Missing code
	body, _ := json.Marshal(discounthttp.ValidateCouponRequest{
		Code: "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := h.ValidateCoupon(w, req)
	assert.Error(t, err)

	// Invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader([]byte("{invalid")))
	w = httptest.NewRecorder()

	err = h.ValidateCoupon(w, req)
	assert.Error(t, err)
}

func TestDiscountHandler_ValidateCoupon_NotFound(t *testing.T) {
	h := setupHandler(&mockRepo{}, time.Now())

	body, _ := json.Marshal(discounthttp.ValidateCouponRequest{
		Code:     "NONEXISTENT",
		Subtotal: 100000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := h.ValidateCoupon(w, req)
	assert.Error(t, err)
}

func TestDiscountHandler_CreateCoupon(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	repo := &mockRepo{}
	h := setupHandler(repo, now)

	body, _ := json.Marshal(discounthttp.CreateCouponRequest{
		Code:          "NEWCOUPON",
		Type:          discountdomain.CouponTypeFixedAmount,
		DiscountValue: 25000,
		MinSpend:      50000,
		QuotaTotal:    50,
		StartsAt:      now.Add(-time.Hour),
		ExpiresAt:     now.Add(24 * time.Hour),
		IsActive:      true,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/coupons", bytes.NewReader(body))
	w := httptest.NewRecorder()

	err := h.CreateCoupon(w, req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp discounthttp.CouponResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "NEWCOUPON", resp.Code)
	assert.Equal(t, int64(25000), resp.DiscountValue)
	assert.Equal(t, 50, resp.QuotaRemaining)
}

func TestDiscountHandler_ListCoupons(t *testing.T) {
	now := time.Now()
	repo := &mockRepo{
		listResult: []discountdomain.Coupon{
			{
				ID:             uuid.New(),
				Code:           "COUPON1",
				Type:           discountdomain.CouponTypeFreeShipping,
				QuotaTotal:     100,
				QuotaRemaining: 100,
				StartsAt:       now,
				ExpiresAt:      now.Add(time.Hour),
				IsActive:       true,
			},
		},
		listTotal: 1,
	}
	h := setupHandler(repo, now)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/coupons?page=1&limit=10&is_active=true", nil)
	w := httptest.NewRecorder()

	err := h.ListCoupons(w, req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp discounthttp.ListCouponsResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "COUPON1", resp.Items[0].Code)
}
