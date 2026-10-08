package usecase

import (
	"context"
	"testing"

	cartDomain "komecore/internal/modules/cart/domain"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"

	"github.com/google/uuid"
)

func TestUpdateItemByID_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	_ = cart.AddItem(productID, shopID, 2, cartDomain.ItemOptions{"size": "small", "color": "blue"})
	cartItemID := cart.Items[0].ID

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 20,
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

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	opt := cartDomain.ItemOptions{"size": "large", "color": "red"}
	err := uc.UpdateItemByID(ctx, UpdateItemByIDInput{
		CustomerID:  customerID,
		CartItemID:  cartItemID,
		Quantity:    4,
		ItemOptions: &opt,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cartR.saved {
		t.Errorf("expected cart to be saved")
	}

	if cart.Items[0].Quantity != 4 {
		t.Errorf("expected quantity 4, got %d", cart.Items[0].Quantity)
	}
	if !cart.Items[0].ItemOptions.Equals(opt) {
		t.Errorf("expected options %+v, got %+v", opt, cart.Items[0].ItemOptions)
	}
}

func TestUpdateItemByID_NilOptions_PreservesExistingOptions(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()
	optLarge := cartDomain.ItemOptions{"size": "large", "color": "red"}

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	_ = cart.AddItem(productID, shopID, 2, optLarge)
	cartItemID := cart.Items[0].ID

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 20,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Papan Bunga",
			Status: productDomain.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	// Update with ItemOptions == nil (quantity-only update)
	err := uc.UpdateItemByID(ctx, UpdateItemByIDInput{
		CustomerID:  customerID,
		CartItemID:  cartItemID,
		Quantity:    6,
		ItemOptions: nil,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cart.Items[0].Quantity != 6 {
		t.Errorf("expected quantity 6, got %d", cart.Items[0].Quantity)
	}
	if !cart.Items[0].ItemOptions.Equals(optLarge) {
		t.Errorf("expected options %+v to be preserved, got %+v", optLarge, cart.Items[0].ItemOptions)
	}
}

func TestUpdateItemByID_MultiStyle_StockExceeded(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	_ = cart.AddItem(productID, shopID, 2, cartDomain.ItemOptions{"size": "small", "color": "blue"})
	_ = cart.AddItem(productID, shopID, 3, cartDomain.ItemOptions{"size": "large", "color": "red"})
	largeItemID := cart.Items[1].ID

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 6, // Total available stock is 6 across all styles
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

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	// Try increasing large item from 3 to 5 (Total in cart would be 2 small + 5 large = 7 > 6)
	err := uc.UpdateItemByID(ctx, UpdateItemByIDInput{
		CustomerID:  customerID,
		CartItemID:  largeItemID,
		Quantity:    5,
		ItemOptions: nil,
	})

	if err == nil {
		t.Fatalf("expected error due to total stock exceeded across styles, got nil")
	}
}

func TestUpdateItemByID_InsufficientStock(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}
	_ = cart.AddItem(productID, shopID, 2)
	cartItemID := cart.Items[0].ID

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 3,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Papan Bunga",
			Status: productDomain.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	err := uc.UpdateItemByID(ctx, UpdateItemByIDInput{
		CustomerID: customerID,
		CartItemID: cartItemID,
		Quantity:   5, // exceeds available stock 3
	})

	if err == nil {
		t.Fatalf("expected error for insufficient stock, got nil")
	}
}

func TestUpdateItemByID_CartItemNotFound(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items:      []cartDomain.CartItem{},
	}

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{}
	prodR := &mockAddItemProdRepo{}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	err := uc.UpdateItemByID(ctx, UpdateItemByIDInput{
		CustomerID: customerID,
		CartItemID: uuid.New(), // random non-existent cart item
		Quantity:   1,
	})

	if err == nil {
		t.Fatalf("expected error for non-existent cart item, got nil")
	}
}

func TestUpdateItem_Execute_NoOptions_PreservesOptions(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()
	optLarge := cartDomain.ItemOptions{"size": "large", "color": "red"}

	cart := &cartDomain.Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []cartDomain.CartItem{
			{
				ID:          uuid.New(),
				ProductID:   productID,
				ShopID:      shopID,
				Quantity:    2,
				ItemOptions: optLarge,
			},
		},
	}

	cartR := &mockSaveCartRepository{cart: cart}
	invR := &mockAddItemInvRepo{
		inv: &inventoryDomain.Inventory{
			ProductID:  productID,
			ShopID:     shopID,
			TotalStock: 20,
		},
	}
	prodR := &mockAddItemProdRepo{
		prod: &productDomain.Product{
			ID:     productID,
			Name:   "Papan Bunga",
			Status: productDomain.ProductStatusActive,
		},
	}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	uc := NewCartService(cartR, invR, prodR, nil, nil, nil, exec, tx)

	// Execute without options (e.g. PUT /{shopID}/{productID})
	err := uc.UpdateItem(ctx, UpdateItemInput{
		CustomerID: customerID,
		ProductID:  productID,
		ShopID:     shopID,
		Quantity:   5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item in cart, got %d (duplicated)", len(cart.Items))
	}
	if cart.Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", cart.Items[0].Quantity)
	}
	if !cart.Items[0].ItemOptions.Equals(optLarge) {
		t.Errorf("expected options %+v to be preserved, got %+v", optLarge, cart.Items[0].ItemOptions)
	}
}
