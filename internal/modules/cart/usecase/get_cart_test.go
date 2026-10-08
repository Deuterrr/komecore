package usecase

import (
	"context"
	"testing"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	cartDomain "komecore/internal/modules/cart/domain"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"
	image "komecore/pkg/imageutil"

	"github.com/google/uuid"
)

// Mocks
type mockCartRepository struct {
	cart *cartDomain.Cart
	err  error
}

func (m *mockCartRepository) GetWithItemsByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cartDomain.Cart, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cart, nil
}

func (m *mockCartRepository) NewCart(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cartDomain.Cart, error) {
	return &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}, nil
}

func (m *mockCartRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	cart *cartDomain.Cart,
) error {
	m.cart = cart
	return nil
}

type mockInventoryRepository struct {
	inventories map[uuid.UUID][]inventoryDomain.Inventory
	err         error
}

func (m *mockInventoryRepository) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventoryDomain.Inventory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.inventories, nil
}

func (m *mockInventoryRepository) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID, shopID uuid.UUID,
) (*inventoryDomain.Inventory, error) {
	return nil, nil
}

type mockProductRepository struct {
	products []productDomain.Product
	err      error
}

func (m *mockProductRepository) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	IDs []uuid.UUID,
) ([]productDomain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.products, nil
}

func (m *mockProductRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*productDomain.Product, error) {
	for i := range m.products {
		if m.products[i].ID == id {
			return &m.products[i], nil
		}
	}
	return nil, nil
}

type mockProductImageRepository struct {
	images map[uuid.UUID][]productDomain.ProductImage
	err    error
}

func (m *mockProductImageRepository) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]productDomain.ProductImage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.images, nil
}

type mockStorageProvider struct {
	storage.Provider
}

func (m *mockStorageProvider) PublicURL(key string, bucket string) string {
	return "http://localhost/" + bucket + "/" + key
}

type mockExecutor struct {
	transaction.Executor
}

// Tests
func TestGetCart_Success_NoDeletedProducts(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()
	shopID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartDomain.CartItem{
			{
				ID:        uuid.New(),
				ProductID: productID1,
				ShopID:    shopID,
				Quantity:  2,
			},
			{
				ID:        uuid.New(),
				ProductID: productID2,
				ShopID:    shopID,
				Quantity:  1,
			},
		},
	}

	products := []productDomain.Product{
		{
			ID:    productID1,
			SKU:   "SKU1",
			Name:  "Product 1",
			Price: 100,
		},
		{
			ID:    productID2,
			SKU:   "SKU2",
			Name:  "Product 2",
			Price: 200,
		},
	}

	inventories := map[uuid.UUID][]inventoryDomain.Inventory{
		productID1: {
			{
				ProductID:     productID1,
				ShopID:        shopID,
				TotalStock:    10,
				ReservedStock: 1,
			},
		},
	}

	images := map[uuid.UUID][]productDomain.ProductImage{
		productID1: {
			{
				ID:        uuid.New(),
				ProductID: productID1,
				Variants: map[image.ResolutionType]productDomain.ImageVariant{
					productDomain.ResolutionThumbnail: {
						Type: productDomain.ResolutionThumbnail,
						Key:  "thumb1",
					},
				},
			},
		},
	}

	cartR := &mockCartRepository{cart: cart}
	invR := &mockInventoryRepository{inventories: inventories}
	prodR := &mockProductRepository{products: products}
	imgR := &mockProductImageRepository{images: images}
	store := &mockStorageProvider{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, imgR, nil, store, exec, nil)
	result, err := uc.GetCart(ctx, customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Cart.Items) != 2 {
		t.Errorf("expected 2 active cart items, got %d", len(result.Cart.Items))
	}

	if len(result.Products) != 2 {
		t.Errorf("expected 2 active products in map, got %d", len(result.Products))
	}

	p1, ok := result.Products[productID1]
	if !ok {
		t.Fatalf("expected product 1 in map")
	}
	if p1.Images.Thumbnail != "http://localhost/public-assets/thumb1" {
		t.Errorf("unexpected thumbnail: %s", p1.Images.Thumbnail)
	}
}

func TestGetCart_Success_WithDeletedProducts(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productIDActive := uuid.New()
	productIDDeleted := uuid.New()
	shopID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartDomain.CartItem{
			{
				ID:        uuid.New(),
				ProductID: productIDActive,
				ShopID:    shopID,
				Quantity:  2,
			},
			{
				ID:        uuid.New(),
				ProductID: productIDDeleted,
				ShopID:    shopID,
				Quantity:  1,
			},
		},
	}

	// In the persistence layer, FindByIDs filters `WHERE deleted_at IS NULL`,
	// so soft-deleted products are excluded from the returned slice.
	products := []productDomain.Product{
		{
			ID:    productIDActive,
			SKU:   "ACTIVE-SKU",
			Name:  "Active Product",
			Price: 100,
		},
	}

	inventories := map[uuid.UUID][]inventoryDomain.Inventory{}
	images := map[uuid.UUID][]productDomain.ProductImage{}

	cartR := &mockCartRepository{cart: cart}
	invR := &mockInventoryRepository{inventories: inventories}
	prodR := &mockProductRepository{products: products}
	imgR := &mockProductImageRepository{images: images}
	store := &mockStorageProvider{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, imgR, nil, store, exec, nil)
	result, err := uc.GetCart(ctx, customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Cart.Items) != 1 {
		t.Errorf("expected 1 active cart item, got %d", len(result.Cart.Items))
	}
	if result.Cart.Items[0].ProductID != productIDActive {
		t.Errorf("expected remaining item to be active product, got %v", result.Cart.Items[0].ProductID)
	}

	if len(result.Products) != 1 {
		t.Errorf("expected 1 product in map, got %d", len(result.Products))
	}
	if _, ok := result.Products[productIDActive]; !ok {
		t.Errorf("expected active product to be in map")
	}
	if _, ok := result.Products[productIDDeleted]; ok {
		t.Errorf("expected soft-deleted product to be filtered out of map")
	}
}

func TestGetCart_Success_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productIDNotFound := uuid.New()
	shopID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartDomain.CartItem{
			{
				ID:        uuid.New(),
				ProductID: productIDNotFound,
				ShopID:    shopID,
				Quantity:  2,
			},
		},
	}

	products := []productDomain.Product{}
	inventories := map[uuid.UUID][]inventoryDomain.Inventory{}
	images := map[uuid.UUID][]productDomain.ProductImage{}

	cartR := &mockCartRepository{cart: cart}
	invR := &mockInventoryRepository{inventories: inventories}
	prodR := &mockProductRepository{products: products}
	imgR := &mockProductImageRepository{images: images}
	store := &mockStorageProvider{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, imgR, nil, store, exec, nil)
	result, err := uc.GetCart(ctx, customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Cart.Items) != 0 {
		t.Errorf("expected 0 cart items (since product was not found), got %d", len(result.Cart.Items))
	}

	if len(result.Products) != 0 {
		t.Errorf("expected 0 products in map, got %d", len(result.Products))
	}
}
