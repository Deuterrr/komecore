package discount

import (
	"context"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type mockCouponRepo struct {
	coupons      map[string]*Coupon
	couponsByID  map[uuid.UUID]*Coupon
	redemptions  map[uuid.UUID]*CouponRedemption // orderID -> redemption
	savedRedemps []CouponRedemption
}

func newMockCouponRepo() *mockCouponRepo {
	return &mockCouponRepo{
		coupons:     make(map[string]*Coupon),
		couponsByID: make(map[uuid.UUID]*Coupon),
		redemptions: make(map[uuid.UUID]*CouponRedemption),
	}
}

func (m *mockCouponRepo) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*Coupon, error) {
	c, exists := m.couponsByID[id]
	if !exists {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *mockCouponRepo) GetByCode(
	ctx context.Context,
	exec transaction.Executor,
	code string,
) (*Coupon, error) {
	c, exists := m.coupons[NormalizeCode(code)]
	if !exists {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *mockCouponRepo) Save(
	ctx context.Context,
	exec transaction.Executor,
	coupon Coupon,
) error {
	m.coupons[coupon.Code] = &coupon
	m.couponsByID[coupon.ID] = &coupon
	return nil
}

func (m *mockCouponRepo) List(
	ctx context.Context,
	exec transaction.Executor,
	params ListCouponsParams,
) ([]Coupon, int, error) {
	var results []Coupon
	for _, c := range m.coupons {
		if params.IsActive != nil && c.IsActive != *params.IsActive {
			continue
		}
		results = append(results, *c)
	}
	return results, len(results), nil
}

func (m *mockCouponRepo) DecrementQuota(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	c, exists := m.couponsByID[id]
	if !exists {
		return ErrCouponNotFound
	}
	if c.QuotaRemaining <= 0 || !c.IsActive {
		return ErrCouponQuotaExceeded
	}
	c.QuotaRemaining--
	return nil
}

func (m *mockCouponRepo) SaveRedemption(
	ctx context.Context,
	exec transaction.Executor,
	redemption CouponRedemption,
) error {
	m.redemptions[redemption.OrderID] = &redemption
	m.savedRedemps = append(m.savedRedemps, redemption)
	return nil
}

func (m *mockCouponRepo) GetRedemptionByOrder(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) (*CouponRedemption, error) {
	r, exists := m.redemptions[orderID]
	if !exists {
		return nil, nil
	}
	return r, nil
}

type mockExecutor struct{}

func (m *mockExecutor) Exec(ctx context.Context, sql string, args ...any) (transaction.Result, error) {
	return transaction.NewResult(1), nil
}

func (m *mockExecutor) Query(ctx context.Context, sql string, args ...any) (transaction.Rows, error) {
	return nil, nil
}

func (m *mockExecutor) QueryRow(ctx context.Context, sql string, args ...any) transaction.Row {
	return nil
}
