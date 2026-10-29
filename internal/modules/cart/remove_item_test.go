package cart

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRemoveItemByID_Success(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()
	cartItemID := uuid.New()

	cart := &Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []CartItem{
			{
				ID:          cartItemID,
				ProductID:   productID,
				ShopID:      shopID,
				Quantity:    2,
				ItemOptions: ItemOptions{"size": "small", "color": "blue"},
			},
		},
	}

	cartR := &mockSaveCartRepository{cart: cart}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	svc := NewService(cartR, nil, nil, nil, nil, nil, exec, tx)

	err := svc.RemoveItemByID(ctx, RemoveItemByIDInput{
		CustomerID: customerID,
		CartItemID: cartItemID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cartR.saved {
		t.Errorf("expected cart to be saved after removal")
	}

	if cart.Items[0].DeletedAt == nil {
		t.Errorf("expected item to be soft-deleted")
	}
}

func TestRemoveItemByID_NotFound_ItemDoesNotExist(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	cart := &Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []CartItem{
			{
				ID:        uuid.New(),
				ProductID: productID,
				ShopID:    shopID,
				Quantity:  1,
			},
		},
	}

	cartR := &mockSaveCartRepository{cart: cart}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	svc := NewService(cartR, nil, nil, nil, nil, nil, exec, tx)

	err := svc.RemoveItemByID(ctx, RemoveItemByIDInput{
		CustomerID: customerID,
		CartItemID: uuid.New(), // non-existent item
	})

	if err == nil {
		t.Errorf("expected error for non-existent item ID, got nil")
	}
}

func TestRemoveItemByID_NotFound_CartDoesNotExist(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()

	cartR := &mockSaveCartRepository{cart: nil}
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	svc := NewService(cartR, nil, nil, nil, nil, nil, exec, tx)

	err := svc.RemoveItemByID(ctx, RemoveItemByIDInput{
		CustomerID: customerID,
		CartItemID: uuid.New(),
	})

	if err == nil {
		t.Errorf("expected error for non-existent cart, got nil")
	}
}

func TestRemoveItem_WithOptions(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	shopID := uuid.New()
	productID := uuid.New()

	optSmall := ItemOptions{"size": "small", "color": "blue"}
	optLarge := ItemOptions{"size": "large", "color": "white"}

	cart := &Cart{
		ID:         uuid.New(),
		CustomerID: customerID,
		Items: []CartItem{
			{
				ID:          uuid.New(),
				ProductID:   productID,
				ShopID:      shopID,
				Quantity:    1,
				ItemOptions: optSmall,
			},
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
	tx := &mockTransactor{}
	exec := &mockExecutor{}

	svc := NewService(cartR, nil, nil, nil, nil, nil, exec, tx)

	// Remove specifically the large item
	err := svc.RemoveItem(ctx, RemoveItemInput{
		CustomerID:  customerID,
		ProductID:   productID,
		ShopID:      shopID,
		ItemOptions: &optLarge,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cart.Items[0].DeletedAt != nil {
		t.Errorf("expected small item to remain active")
	}
	if cart.Items[1].DeletedAt == nil {
		t.Errorf("expected large item to be soft-deleted")
	}
}
