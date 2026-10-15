package cartusecase

import (
	"context"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"

	"github.com/google/uuid"
)

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

type mockCartShopRepo struct {
	shop *shop.Shop
	err  error
}

func (m *mockCartShopRepo) FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]shop.Shop, error) {
	return nil, nil
}

func (m *mockCartShopRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*shop.Shop, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.shop != nil {
		return m.shop, nil
	}
	return &shop.Shop{
		ID:             id,
		Name:           "Mock Shop",
		Slug:           "mock-shop",
		IsActive:       true,
		ApprovalStatus: shop.ShopApprovalStatusApproved,
	}, nil
}

type mockSaveCartRepository struct {
	cart    *cart.Cart
	saved   bool
	saveErr error
	getErr  error
}

func (m *mockSaveCartRepository) GetWithItemsByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cart.Cart, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.cart, nil
}

func (m *mockSaveCartRepository) NewCart(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cart.Cart, error) {
	return &cart.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cart.CartItem{},
	}, nil
}

func (m *mockSaveCartRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	cart *cart.Cart,
) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.cart = cart
	m.saved = true
	return nil
}

type mockAddItemInvRepo struct {
	inv *inventory.Inventory
}

func (m *mockAddItemInvRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventory.Inventory, error) {
	return nil, nil
}

func (m *mockAddItemInvRepo) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID, shopID uuid.UUID,
) (*inventory.Inventory, error) {
	return m.inv, nil
}

type mockAddItemProdRepo struct {
	prod *product.Product
}

func (m *mockAddItemProdRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	ids []uuid.UUID,
) ([]product.Product, error) {
	return nil, nil
}

func (m *mockAddItemProdRepo) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*product.Product, error) {
	return m.prod, nil
}

func TestAddItem_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{
		cart: &cart.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cart.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Operable Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventory.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &product.Product{
			ID:     productID,
			Name:   "Cotton T-Shirt",
			Status: product.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	err := uc.AddItem(ctx, AddItemInput{
		CustomerID: customerID,
		ShopID:     shopID,
		ProductID:  productID,
		Quantity:   2,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cartR.saved {
		t.Errorf("expected cart to be saved")
	}
}

func TestAddItem_RejectInactiveShop(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Inactive Shop",
			IsActive:       false,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{}
	prodR := &mockAddItemProdRepo{}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	err := uc.AddItem(ctx, AddItemInput{
		CustomerID: customerID,
		ShopID:     shopID,
		ProductID:  productID,
		Quantity:   1,
	})

	if err == nil {
		t.Errorf("expected error for inactive shop, got nil")
	}
}

func TestAddItem_RejectPendingShop(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Pending Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusPending,
		},
	}
	invR := &mockAddItemInvRepo{}
	prodR := &mockAddItemProdRepo{}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	err := uc.AddItem(ctx, AddItemInput{
		CustomerID: customerID,
		ShopID:     shopID,
		ProductID:  productID,
		Quantity:   1,
	})

	if err == nil {
		t.Errorf("expected error for pending shop, got nil")
	}
}

func TestAddItem_RejectInactiveProduct(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{
		cart: &cart.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cart.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventory.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &product.Product{
			ID:     productID,
			Name:   "Inactive Item",
			Status: product.ProductStatusInactive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	err := uc.AddItem(ctx, AddItemInput{
		CustomerID: customerID,
		ShopID:     shopID,
		ProductID:  productID,
		Quantity:   1,
	})

	if err == nil {
		t.Errorf("expected error for inactive product, got nil")
	}
}

func TestAddItem_RejectArchivedProduct(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{
		cart: &cart.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cart.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventory.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &product.Product{
			ID:     productID,
			Name:   "Archived Item",
			Status: product.ProductStatusArchived,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	err := uc.AddItem(ctx, AddItemInput{
		CustomerID: customerID,
		ShopID:     shopID,
		ProductID:  productID,
		Quantity:   1,
	})

	if err == nil {
		t.Errorf("expected error (not found) for archived product, got nil")
	}
}

func TestAddItem_MultiStyle_StockExceeded(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cart.CartItem{},
	}
	// Already has 3 of Small in cart
	_ = c.AddItem(productID, shopID, 3, cart.ItemOptions{"size": "small", "color": "blue"})

	cartR := &mockSaveCartRepository{cart: c}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventory.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 5, // Total stock available is 5
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &product.Product{
			ID:     productID,
			Name:   "Mechanical Keyboard",
			Status: product.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	// Try adding 3 of Large (Total across styles would be 3 small + 3 large = 6 > 5)
	err := uc.AddItem(ctx, AddItemInput{
		CustomerID:  customerID,
		ShopID:      shopID,
		ProductID:   productID,
		Quantity:    3,
		ItemOptions: cart.ItemOptions{"size": "large", "color": "red"},
	})

	if err == nil {
		t.Fatalf("expected error for exceeding total product stock across styles, got nil")
	}
}

func TestAddItem_MultiStyle_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cart.CartItem{},
	}
	// Already has 2 of Small in cart
	_ = c.AddItem(productID, shopID, 2, cart.ItemOptions{"size": "small", "color": "blue"})

	cartR := &mockSaveCartRepository{cart: c}
	shopR := &mockCartShopRepo{
		shop: &shop.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shop.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventory.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 5,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &product.Product{
			ID:     productID,
			Name:   "Mechanical Keyboard",
			Status: product.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, shopR, nil, exec, tx)

	// Add 2 of Large (Total across styles = 2 + 2 = 4 <= 5)
	err := uc.AddItem(ctx, AddItemInput{
		CustomerID:  customerID,
		ShopID:      shopID,
		ProductID:   productID,
		Quantity:    2,
		ItemOptions: cart.ItemOptions{"size": "large", "color": "red"},
	})

	if err != nil {
		t.Fatalf("unexpected error adding second style: %v", err)
	}

	if len(c.Items) != 2 {
		t.Fatalf("expected 2 separate style items in cart, got %d", len(c.Items))
	}
	if c.Items[0].Quantity != 2 || c.Items[1].Quantity != 2 {
		t.Errorf("expected quantities (2, 2), got (%d, %d)", c.Items[0].Quantity, c.Items[1].Quantity)
	}
}
