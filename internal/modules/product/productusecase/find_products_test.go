package productusecase

import (
	"context"
	"testing"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/shop/shopdomain"
	"komecore/internal/modules/shop/shoprepo"

	"github.com/google/uuid"
)

type mockFindProductRepo struct {
	productrepo.ProductRepository
	capturedParams productrepo.FindProductParams
	products       []productdomain.ProductWithInventory
}

func (m *mockFindProductRepo) FindProductsWithInventory(
	ctx context.Context,
	exec transaction.Executor,
	params productrepo.FindProductParams,
) ([]productdomain.ProductWithInventory, int, error) {
	m.capturedParams = params
	return m.products, len(m.products), nil
}

type mockFindInventoryRepo struct {
	inventoryrepo.InventoryRepository
}

func (m *mockFindInventoryRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	return map[uuid.UUID][]inventorydomain.Inventory{}, nil
}

type mockFindImgRepo struct {
	productrepo.ProductImageRepository
}

func (m *mockFindImgRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]productdomain.ProductImage, error) {
	return map[uuid.UUID][]productdomain.ProductImage{}, nil
}

type mockFindShopRepo struct {
	shoprepo.ShopRepository
}

func (m *mockFindShopRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	IDs []uuid.UUID,
) ([]shopdomain.Shop, error) {
	return []shopdomain.Shop{}, nil
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

	uc := NewFindProductsUsecase(productRepo, invRepo, imgRepo, shopRepo, fileStore, exec)

	// Case 1: Query with ShopID
	_, _, err := uc.Execute(ctx, FindProductsInput{
		ShopID: &shopIDStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopID == nil || *productRepo.capturedParams.ShopID != shopID {
		t.Errorf("expected ShopID %v, got %v", shopID, productRepo.capturedParams.ShopID)
	}

	// Case 2: Query with ShopSlug
	_, _, err = uc.Execute(ctx, FindProductsInput{
		ShopSlug: &shopSlug,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopSlug == nil || *productRepo.capturedParams.ShopSlug != shopSlug {
		t.Errorf("expected ShopSlug %v, got %v", shopSlug, productRepo.capturedParams.ShopSlug)
	}

	// Case 3: Query without Shop filter (legacy backward compatibility)
	_, _, err = uc.Execute(ctx, FindProductsInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productRepo.capturedParams.ShopID != nil || productRepo.capturedParams.ShopSlug != nil {
		t.Errorf("expected nil shop filters for legacy query, got shopID: %v, shopSlug: %v",
			productRepo.capturedParams.ShopID, productRepo.capturedParams.ShopSlug)
	}
}

type mockCustomInvRepo struct {
	inventoryrepo.InventoryRepository
	inventories map[uuid.UUID][]inventorydomain.Inventory
}

func (m *mockCustomInvRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	return m.inventories, nil
}

type mockCustomShopRepo struct {
	shoprepo.ShopRepository
	shops []shopdomain.Shop
}

func (m *mockCustomShopRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	IDs []uuid.UUID,
) ([]shopdomain.Shop, error) {
	return m.shops, nil
}

func TestFindProducts_FiltersInactiveOrUnapprovedShops(t *testing.T) {
	ctx := context.Background()
	prodID := uuid.New()
	activeShopID := uuid.New()
	inactiveShopID := uuid.New()
	pendingShopID := uuid.New()

	productRepo := &mockFindProductRepo{
		products: []productdomain.ProductWithInventory{
			{
				Product: productdomain.Product{
					ID:     prodID,
					Name:   "Wireless Mouse",
					Status: productdomain.ProductStatusActive,
				},
			},
		},
	}

	invRepo := &mockCustomInvRepo{
		inventories: map[uuid.UUID][]inventorydomain.Inventory{
			prodID: {
				{ShopID: activeShopID, TotalStock: 10},
				{ShopID: inactiveShopID, TotalStock: 5},
				{ShopID: pendingShopID, TotalStock: 8},
			},
		},
	}

	customShopRepo := &mockCustomShopRepo{
		shops: []shopdomain.Shop{
			{
				ID:             activeShopID,
				Name:           "Active Shop",
				Slug:           "active-shop",
				IsActive:       true,
				ApprovalStatus: shopdomain.ShopApprovalStatusApproved,
			},
			{
				ID:             inactiveShopID,
				Name:           "Inactive Shop",
				Slug:           "inactive-shop",
				IsActive:       false,
				ApprovalStatus: shopdomain.ShopApprovalStatusApproved,
			},
			{
				ID:             pendingShopID,
				Name:           "Pending Shop",
				Slug:           "pending-shop",
				IsActive:       true,
				ApprovalStatus: shopdomain.ShopApprovalStatusPending,
			},
		},
	}

	imgRepo := &mockFindImgRepo{}
	fileStore := &mockFileStore{}
	exec := &mockExecutor{}

	uc := NewFindProductsUsecase(productRepo, invRepo, imgRepo, customShopRepo, fileStore, exec)

	res, _, err := uc.Execute(ctx, FindProductsInput{})
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
