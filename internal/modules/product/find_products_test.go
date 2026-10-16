package product

import (
	"context"
	"testing"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/shop"

	"github.com/google/uuid"
)

type mockFindProductRepo struct {
	Repository
	capturedParams FindProductParams
	products       []ProductWithInventory
}

func (m *mockFindProductRepo) FindProductsWithInventory(
	ctx context.Context,
	exec transaction.Executor,
	params FindProductParams,
) ([]ProductWithInventory, int, error) {
	m.capturedParams = params
	return m.products, len(m.products), nil
}

type mockFindInventoryRepo struct {
	inventory.Repository
}

func (m *mockFindInventoryRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventory.Inventory, error) {
	return map[uuid.UUID][]inventory.Inventory{}, nil
}

type mockFindImgRepo struct {
	ImageRepository
}

func (m *mockFindImgRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]ProductImage, error) {
	return map[uuid.UUID][]ProductImage{}, nil
}

type mockFindShopRepo struct {
	shop.Repository
}

func (m *mockFindShopRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	IDs []uuid.UUID,
) ([]shop.Shop, error) {
	return []shop.Shop{}, nil
}

type mockFileStore struct {
	storage.Provider
}

func (m *mockFileStore) PublicURL(key, bucket string) string {
	return "http://localhost/" + key
}

func TestFindProducts_WithShopFilter(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	shopIDStr := shopID.String()
	shopSlug := "central-store"

	productRepo := &mockFindProductRepo{}
	invRepo := &mockFindInventoryRepo{}
	imgRepo := &mockFindImgRepo{}
	shopRepo := &mockFindShopRepo{}
	fileStore := &mockFileStore{}
	exec := &mockExecutor{}

	svc := NewService(exec, nil, productRepo, invRepo, imgRepo, nil, shopRepo, nil, nil, fileStore)

	// Case 1: Query with ShopID
	_, _, err := svc.FindProducts(ctx, FindProductsInput{
		ShopID: &shopIDStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopID == nil || *productRepo.capturedParams.ShopID != shopID {
		t.Errorf("expected ShopID %v, got %v", shopID, productRepo.capturedParams.ShopID)
	}

	// Case 2: Query with ShopSlug
	_, _, err = svc.FindProducts(ctx, FindProductsInput{
		ShopSlug: &shopSlug,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopSlug == nil || *productRepo.capturedParams.ShopSlug != shopSlug {
		t.Errorf("expected ShopSlug %v, got %v", shopSlug, productRepo.capturedParams.ShopSlug)
	}

	// Case 3: Query without Shop filter (legacy backward compatibility)
	_, _, err = svc.FindProducts(ctx, FindProductsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopID != nil || productRepo.capturedParams.ShopSlug != nil {
		t.Errorf("expected nil shop filters for legacy query, got shopID: %v, shopSlug: %v",
			productRepo.capturedParams.ShopID, productRepo.capturedParams.ShopSlug)
	}
}

type mockCustomInvRepo struct {
	inventory.Repository
	inventories map[uuid.UUID][]inventory.Inventory
}

func (m *mockCustomInvRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventory.Inventory, error) {
	return m.inventories, nil
}

type mockCustomShopRepo struct {
	shop.Repository
	shops []shop.Shop
}

func (m *mockCustomShopRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	IDs []uuid.UUID,
) ([]shop.Shop, error) {
	return m.shops, nil
}

func TestFindProducts_FiltersInactiveOrUnapprovedShops(t *testing.T) {
	ctx := context.Background()
	prodID := uuid.New()
	activeShopID := uuid.New()
	inactiveShopID := uuid.New()
	pendingShopID := uuid.New()

	productRepo := &mockFindProductRepo{
		products: []ProductWithInventory{
			{
				Product: Product{
					ID:     prodID,
					Name:   "Wireless Mouse",
					Status: ProductStatusActive,
				},
			},
		},
	}

	invRepo := &mockCustomInvRepo{
		inventories: map[uuid.UUID][]inventory.Inventory{
			prodID: {
				{ShopID: activeShopID, TotalStock: 10},
				{ShopID: inactiveShopID, TotalStock: 5},
				{ShopID: pendingShopID, TotalStock: 8},
			},
		},
	}

	customShopRepo := &mockCustomShopRepo{
		shops: []shop.Shop{
			{
				ID:             activeShopID,
				Name:           "Active Shop",
				Slug:           "active-shop",
				IsActive:       true,
				ApprovalStatus: shop.ShopApprovalStatusApproved,
			},
			{
				ID:             inactiveShopID,
				Name:           "Inactive Shop",
				Slug:           "inactive-shop",
				IsActive:       false,
				ApprovalStatus: shop.ShopApprovalStatusApproved,
			},
			{
				ID:             pendingShopID,
				Name:           "Pending Shop",
				Slug:           "pending-shop",
				IsActive:       true,
				ApprovalStatus: shop.ShopApprovalStatusPending,
			},
		},
	}

	imgRepo := &mockFindImgRepo{}
	fileStore := &mockFileStore{}
	exec := &mockExecutor{}

	svc := NewService(exec, nil, productRepo, invRepo, imgRepo, nil, customShopRepo, nil, nil, fileStore)

	res, _, err := svc.FindProducts(ctx, FindProductsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 product result, got %d", len(res))
	}

	// Total stock should ONLY count active + approved shop (10), not inactive (5) or pending (8)
	if res[0].Inventory.TotalStock != 10 {
		t.Errorf("expected TotalStock to be 10, got %d", res[0].Inventory.TotalStock)
	}

	if len(res[0].Availability) != 1 {
		t.Errorf("expected 1 available shop, got %d", len(res[0].Availability))
	} else if res[0].Availability[0].ShopName != "Active Shop" {
		t.Errorf("expected Active Shop in availability, got %s", res[0].Availability[0].ShopName)
	}
}

func TestFindProducts_FullTextSearch(t *testing.T) {
	ctx := context.Background()
	productRepo := &mockFindProductRepo{}
	invRepo := &mockFindInventoryRepo{}
	imgRepo := &mockFindImgRepo{}
	shopRepo := &mockFindShopRepo{}
	fileStore := &mockFileStore{}
	exec := &mockExecutor{}

	svc := NewService(exec, nil, productRepo, invRepo, imgRepo, nil, shopRepo, nil, nil, fileStore)

	searchQuery := "wireless earbuds"
	_, _, err := svc.FindProducts(ctx, FindProductsInput{
		SearchQuery: &searchQuery,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if productRepo.capturedParams.SearchQuery == nil || *productRepo.capturedParams.SearchQuery != searchQuery {
		t.Errorf("expected captured SearchQuery to be %q, got %v", searchQuery, productRepo.capturedParams.SearchQuery)
	}

	if len(productRepo.capturedParams.Sorts) == 0 {
		t.Fatalf("expected sorts to be populated")
	}

	if productRepo.capturedParams.Sorts[0].By != ProductSortRelevance {
		t.Errorf("expected default sort to be relevance when search query provided, got %v", productRepo.capturedParams.Sorts[0].By)
	}
}
