package usecase

import (
	"context"
	"testing"

	transaction "komecore/internal/infra/transactor"
	cartDomain "komecore/internal/modules/cart/domain"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"
	shopDomain "komecore/internal/modules/shop/domain"

	"github.com/google/uuid"
)

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

type mockCartShopRepo struct {
	shop *shopDomain.Shop
	err  error
}

func (m *mockCartShopRepo) FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]shopDomain.Shop, error) {
	return nil, nil
}

func (m *mockCartShopRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*shopDomain.Shop, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.shop != nil {
		return m.shop, nil
	}
	return &shopDomain.Shop{
		ID:             id,
		Name:           "Mock Shop",
		Slug:           "mock-shop",
		IsActive:       true,
		ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
	}, nil
}

type mockSaveCartRepository struct {
	cart    *cartDomain.Cart
	saved   bool
	saveErr error
	getErr  error
}

func (m *mockSaveCartRepository) GetWithItemsByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cartDomain.Cart, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.cart, nil
}

func (m *mockSaveCartRepository) NewCart(
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

func (m *mockSaveCartRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	cart *cartDomain.Cart,
) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.cart = cart
	m.saved = true
	return nil
}

type mockAddItemInvRepo struct {
	inv *inventoryDomain.Inventory
}

func (m *mockAddItemInvRepo) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventoryDomain.Inventory, error) {
	return nil, nil
}

func (m *mockAddItemInvRepo) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID, shopID uuid.UUID,
) (*inventoryDomain.Inventory, error) {
	return m.inv, nil
}

type mockAddItemProdRepo struct {
	prod *productDomain.Product
}

func (m *mockAddItemProdRepo) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	ids []uuid.UUID,
) ([]productDomain.Product, error) {
	return nil, nil
}

func (m *mockAddItemProdRepo) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*productDomain.Product, error) {
	return m.prod, nil
}

func TestAddItem_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cartR := &mockSaveCartRepository{
		cart: &cartDomain.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cartDomain.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Operable Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Cotton T-Shirt",
			Status: productDomain.ProductStatusActive,
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
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Inactive Shop",
			IsActive:       false,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
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
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Pending Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusPending,
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
		cart: &cartDomain.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cartDomain.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Inactive Item",
			Status: productDomain.ProductStatusInactive,
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
		cart: &cartDomain.Cart{
			ID:         uuid.New(),
			CustomerID: customerID,
			Items:      []cartDomain.CartItem{},
		},
	}
	shopR := &mockCartShopRepo{
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 10,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Archived Item",
			Status: productDomain.ProductStatusArchived,
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

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	// Already has 3 of Small in cart
	_ = cart.AddItem(productID, shopID, 3, cartDomain.ItemOptions{"size": "small", "color": "blue"})

	cartR := &mockSaveCartRepository{cart: cart}
	shopR := &mockCartShopRepo{
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 5, // Total stock available is 5
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Mechanical Keyboard",
			Status: productDomain.ProductStatusActive,
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
		ItemOptions: cartDomain.ItemOptions{"size": "large", "color": "red"},
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

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	// Already has 2 of Small in cart
	_ = cart.AddItem(productID, shopID, 2, cartDomain.ItemOptions{"size": "small", "color": "blue"})

	cartR := &mockSaveCartRepository{cart: cart}
	shopR := &mockCartShopRepo{
		shop: &shopDomain.Shop{
			ID:             shopID,
			Name:           "Active Shop",
			IsActive:       true,
			ApprovalStatus: shopDomain.ShopApprovalStatusApproved,
		},
	}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 5,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Mechanical Keyboard",
			Status: productDomain.ProductStatusActive,
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
		ItemOptions: cartDomain.ItemOptions{"size": "large", "color": "red"},
	})

	if err != nil {
		t.Fatalf("unexpected error adding second style: %v", err)
	}

	if len(cart.Items) != 2 {
		t.Fatalf("expected 2 separate style items in cart, got %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 2 || cart.Items[1].Quantity != 2 {
		t.Errorf("expected quantities (2, 2), got (%d, %d)", cart.Items[0].Quantity, cart.Items[1].Quantity)
	}
}
