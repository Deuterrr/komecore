package shipmentusecase

import (
	"context"
	"testing"

	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shipping provider mock
// ---------------------------------------------------------------------------

type mockShippingProvider struct {
	rates []shipping.RateOption
	err   error
}

func (m *mockShippingProvider) CalculateRates(ctx context.Context, input shipping.CalculateRatesInput) ([]shipping.RateOption, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rates, nil
}

func (m *mockShippingProvider) CalculateShippingFee(weightKg float64, courierCode string) (int64, error) {
	return 10000, nil
}

// ---------------------------------------------------------------------------
// CourierRepository mock
// ---------------------------------------------------------------------------

type mockCourierRepo struct {
	codes []string
	err   error
}

func (m *mockCourierRepo) ListAll(ctx context.Context, exec transaction.Executor) ([]string, error) {
	return m.codes, m.err
}

func (m *mockCourierRepo) GetActiveCodes(ctx context.Context, exec transaction.Executor, codes []string) ([]string, error) {
	return m.codes, m.err
}

func (m *mockCourierRepo) ValidateCouriers(ctx context.Context, exec transaction.Executor, codes []string) ([]string, error) {
	return m.codes, m.err
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestEstimate_InvalidWeight(t *testing.T) {
	u := NewEstimateShippingOptionsUsecase(
		&mockShippingProvider{},
		&mockExecutor{},
		&mockCourierRepo{codes: []string{"jne", "jnt"}},
	)

	_, err := u.Execute(context.Background(), EstimateShippingOptionsInput{
		ShopID:      uuid.New(),
		Origin:      1,
		Destination: 2,
		Weight:      0,
	})

	if err == nil {
		t.Fatal("expected error for non-positive weight, got nil")
	}
}

func TestEstimate_HappyPath(t *testing.T) {
	shipProv := shipping.NewSimpleProvider()

	u := NewEstimateShippingOptionsUsecase(
		shipProv,
		&mockExecutor{},
		&mockCourierRepo{codes: []string{"jne", "jnt"}},
	)

	res, err := u.Execute(context.Background(), EstimateShippingOptionsInput{
		ShopID:      uuid.New(),
		Origin:      1,
		Destination: 2,
		Weight:      1000,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}

	if res[0].Code != "jne" {
		t.Errorf("expected courier code 'jne', got %s", res[0].Code)
	}
}

func TestEstimate_PriceFilter(t *testing.T) {
	shipProv := shipping.NewSimpleProvider()

	u := NewEstimateShippingOptionsUsecase(
		shipProv,
		&mockExecutor{},
		&mockCourierRepo{codes: []string{"jne", "jnt"}},
	)

	cheapest := "cheapest"
	res, err := u.Execute(context.Background(), EstimateShippingOptionsInput{
		ShopID:      uuid.New(),
		Origin:      1,
		Destination: 2,
		Weight:      1000,
		PriceFilter: &cheapest,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 result for cheapest filter, got %d", len(res))
	}
	if res[0].Cost != 12000 {
		t.Errorf("expected cost 12000, got %d", res[0].Cost)
	}
}
