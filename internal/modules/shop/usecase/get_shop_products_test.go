package usecase

import (
	"context"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type mockShopProductProvider struct {
	results []ShopProductResult
	err     error
}

func (m *mockShopProductProvider) GetShopProducts(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) ([]ShopProductResult, error) {
	return m.results, m.err
}


func TestGetShopProducts_Success(t *testing.T) {
	shopID := uuid.New()
	prodID := uuid.New()
	provider := &mockShopProductProvider{
		results: []ShopProductResult{
			{
				Product: ShopProductInfo{
					ID:        prodID,
					SKU:       "SKU-1",
					Name:      "Product 1",
					Slug:      "product-1",
					Status:    "ACTIVE",
					Price:     10000,
					CreatedAt: time.Now(),
				},
				Inventory: ShopProductInventoryInfo{
					TotalStock:    10,
					ReservedStock: 2,
				},
			},
		},
	}
	exec := &mockExecutor{}

	uc := NewShopService(nil, nil, provider, nil, exec)
	results, err := uc.GetShopProducts(context.Background(), shopID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Product.ID != prodID {
		t.Errorf("expected product ID %v, got %v", prodID, results[0].Product.ID)
	}
	if results[0].Inventory.Available() != 8 {
		t.Errorf("expected available 8, got %d", results[0].Inventory.Available())
	}
}
