package cartdomain_test

import (
	"testing"

	"komecore/internal/modules/cart/cartdomain"

	"github.com/google/uuid"
)

func TestCart_UpdateItemByID_QuantityOnly(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	opt := cartdomain.ItemOptions{"size": "medium", "color": "black"}

	err := cart.AddItem(productID, shopID, 2, opt)
	if err != nil {
		t.Fatalf("unexpected error adding item: %v", err)
	}

	targetID := cart.Items[0].ID

	// Update quantity only with same options
	err = cart.UpdateItemByID(targetID, 5, opt)
	if err != nil {
		t.Fatalf("unexpected error updating item: %v", err)
	}

	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", cart.Items[0].Quantity)
	}
	if !cart.Items[0].ItemOptions.Equals(opt) {
		t.Errorf("expected options %+v, got %+v", opt, cart.Items[0].ItemOptions)
	}
}

func TestCart_UpdateItemByID_OptionsChange_NoCollision(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	optOld := cartdomain.ItemOptions{"size": "small", "color": "blue"}
	optNew := cartdomain.ItemOptions{"size": "large", "color": "red"}

	_ = cart.AddItem(productID, shopID, 1, optOld)
	targetID := cart.Items[0].ID

	err := cart.UpdateItemByID(targetID, 3, optNew)
	if err != nil {
		t.Fatalf("unexpected error updating item options: %v", err)
	}

	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", cart.Items[0].Quantity)
	}
	if !cart.Items[0].ItemOptions.Equals(optNew) {
		t.Errorf("expected options %+v, got %+v", optNew, cart.Items[0].ItemOptions)
	}
}

func TestCart_UpdateItemByID_OptionsChange_WithCollision(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	optA := cartdomain.ItemOptions{"size": "small", "color": "blue"}
	optB := cartdomain.ItemOptions{"size": "large", "color": "red"}

	_ = cart.AddItem(productID, shopID, 2, optA)
	itemAID := cart.Items[0].ID

	_ = cart.AddItem(productID, shopID, 3, optB)
	itemBID := cart.Items[1].ID

	// Change item A's options to optB (collision with item B)
	err := cart.UpdateItemByID(itemAID, 2, optB)
	if err != nil {
		t.Fatalf("unexpected error updating with collision: %v", err)
	}

	// Item A should be soft-deleted, Item B should have qty 2 + 3 = 5
	var itemA, itemB *cartdomain.CartItem
	for i := range cart.Items {
		if cart.Items[i].ID == itemAID {
			itemA = &cart.Items[i]
		}
		if cart.Items[i].ID == itemBID {
			itemB = &cart.Items[i]
		}
	}

	if itemA == nil || itemA.DeletedAt == nil {
		t.Errorf("expected item A to be soft-deleted")
	}
	if itemB == nil || itemB.Quantity != 5 {
		t.Errorf("expected item B quantity to be 5, got %v", itemB)
	}
}

func TestCart_UpdateItemByID_NotFound(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	err := cart.UpdateItemByID(uuid.New(), 1, cartdomain.ItemOptions{})
	if err != cartdomain.ErrCartItemNotFound {
		t.Errorf("expected ErrCartItemNotFound, got %v", err)
	}
}

func TestCart_UpdateItemByID_InvalidQuantity(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	pid := uuid.New()
	_ = cart.AddItem(pid, uuid.New(), 1)
	targetID := cart.Items[0].ID

	err := cart.UpdateItemByID(targetID, 0)
	if err != cartdomain.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for 0, got %v", err)
	}

	err = cart.UpdateItemByID(targetID, -1)
	if err != cartdomain.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for -1, got %v", err)
	}
}

func TestCart_SetItem_NoOptions_UpdatesExistingItemInPlace(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	customOpt := cartdomain.ItemOptions{"size": "large", "color": "white"}

	// Add item with non-default options
	_ = cart.AddItem(productID, shopID, 2, customOpt)
	targetID := cart.Items[0].ID

	// SetItem without options (as happens when PUT /carts/items/{shopID}/{productID} is called)
	err := cart.SetItem(productID, shopID, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should update existing item in-place without duplicating
	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d (item was duplicated)", len(cart.Items))
	}
	if cart.Items[0].ID != targetID {
		t.Errorf("expected item ID to remain %v, got %v", targetID, cart.Items[0].ID)
	}
	if cart.Items[0].Quantity != 7 {
		t.Errorf("expected quantity 7, got %d", cart.Items[0].Quantity)
	}
	// ItemOptions should be preserved
	if !cart.Items[0].ItemOptions.Equals(customOpt) {
		t.Errorf("expected options %+v to be preserved, got %+v", customOpt, cart.Items[0].ItemOptions)
	}
}

func TestCart_RemoveItemByID(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()

	_ = cart.AddItem(productID, shopID, 1, cartdomain.ItemOptions{"size": "small", "color": "blue"})
	_ = cart.AddItem(productID, shopID, 2, cartdomain.ItemOptions{"size": "large", "color": "white"})
	item1ID := cart.Items[0].ID
	item2ID := cart.Items[1].ID

	// Remove item 2 specifically by its ID
	removed := cart.RemoveItemByID(item2ID)
	if !removed {
		t.Fatalf("expected RemoveItemByID to return true")
	}

	if cart.Items[0].DeletedAt != nil {
		t.Errorf("expected item 1 to remain active (not deleted)")
	}
	if cart.Items[1].DeletedAt == nil {
		t.Errorf("expected item 2 to be soft-deleted")
	}

	// Remove item 1
	removed = cart.RemoveItemByID(item1ID)
	if !removed {
		t.Fatalf("expected RemoveItemByID to return true for item 1")
	}
	if cart.Items[0].DeletedAt == nil {
		t.Errorf("expected item 1 to be soft-deleted")
	}

	// Non-existent ID returns false
	if cart.RemoveItemByID(uuid.New()) {
		t.Errorf("expected RemoveItemByID to return false for non-existent item")
	}
}

func TestCart_RemoveItem_WithOptions(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	optSmall := cartdomain.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cartdomain.ItemOptions{"size": "large", "color": "white"}

	_ = cart.AddItem(productID, shopID, 1, optSmall)
	_ = cart.AddItem(productID, shopID, 2, optLarge)

	// Remove item matching optLarge specifically
	removed := cart.RemoveItem(productID, shopID, optLarge)
	if !removed {
		t.Fatalf("expected RemoveItem with options to return true")
	}

	if cart.Items[0].DeletedAt != nil {
		t.Errorf("expected first item (optSmall) to remain active")
	}
	if cart.Items[1].DeletedAt == nil {
		t.Errorf("expected second item (optLarge) to be deleted")
	}
}

func TestCart_SameProduct_DifferentStyles_CoexistAndModify(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	optSmall := cartdomain.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cartdomain.ItemOptions{"size": "large", "color": "red"}

	_ = cart.AddItem(productID, shopID, 2, optSmall)
	_ = cart.AddItem(productID, shopID, 3, optLarge)

	if len(cart.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(cart.Items))
	}
	smallItemID := cart.Items[0].ID
	largeItemID := cart.Items[1].ID

	// Update large item quantity only (no options passed)
	err := cart.UpdateItemByID(largeItemID, 5)
	if err != nil {
		t.Fatalf("unexpected error updating large item: %v", err)
	}

	// Ensure large item updated in place and small item is untouched
	if len(cart.Items) != 2 {
		t.Fatalf("expected still 2 items, got %d", len(cart.Items))
	}
	if cart.Items[0].ID != smallItemID || cart.Items[0].Quantity != 2 || !cart.Items[0].ItemOptions.Equals(optSmall) {
		t.Errorf("small item was unexpectedly modified: %+v", cart.Items[0])
	}
	if cart.Items[1].ID != largeItemID || cart.Items[1].Quantity != 5 || !cart.Items[1].ItemOptions.Equals(optLarge) {
		t.Errorf("large item options or quantity unexpected: %+v", cart.Items[1])
	}
}

func TestCart_ChangeItemShop_DifferentStyles_DoNotMerge(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shop1 := uuid.New()
	shop2 := uuid.New()

	optSmall := cartdomain.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cartdomain.ItemOptions{"size": "large", "color": "red"}

	// Shop 2 has Small
	_ = cart.AddItem(productID, shop2, 2, optSmall)
	// Shop 1 has Large
	_ = cart.AddItem(productID, shop1, 3, optLarge)
	largeItemID := cart.Items[1].ID

	// Move large item from Shop 1 to Shop 2
	ok := cart.ChangeItemShop(largeItemID, shop2)
	if !ok {
		t.Fatalf("expected ChangeItemShop to succeed")
	}

	// Both items should be active in Shop 2 as separate items (not merged into Small)
	if cart.Items[0].DeletedAt != nil {
		t.Errorf("expected Small item to remain active")
	}
	if cart.Items[1].DeletedAt != nil {
		t.Errorf("expected Large item to remain active (not soft-deleted)")
	}
	if cart.Items[1].ShopID != shop2 {
		t.Errorf("expected Large item shop_id to be shop2")
	}
	if cart.Items[0].Quantity != 2 || cart.Items[1].Quantity != 3 {
		t.Errorf("expected quantities 2 and 3, got %d and %d", cart.Items[0].Quantity, cart.Items[1].Quantity)
	}
}

func TestCart_ChangeItemShop_SameStyle_Merges(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shop1 := uuid.New()
	shop2 := uuid.New()

	optLarge := cartdomain.ItemOptions{"size": "large", "color": "red"}

	// Shop 2 has Large (qty 2)
	_ = cart.AddItem(productID, shop2, 2, optLarge)
	// Shop 1 has Large (qty 3)
	_ = cart.AddItem(productID, shop1, 3, optLarge)
	shop1LargeItemID := cart.Items[1].ID

	// Move Large item from Shop 1 to Shop 2 (same product and same style)
	ok := cart.ChangeItemShop(shop1LargeItemID, shop2)
	if !ok {
		t.Fatalf("expected ChangeItemShop to succeed")
	}

	// Should merge into Shop 2 Large item (qty 5) and soft-delete Shop 1 item
	if cart.Items[0].Quantity != 5 {
		t.Errorf("expected merged quantity 5, got %d", cart.Items[0].Quantity)
	}
	if cart.Items[1].DeletedAt == nil {
		t.Errorf("expected moved item to be soft-deleted after merge")
	}
}

func TestCart_TotalProductQuantity(t *testing.T) {
	cart := &cartdomain.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()

	_ = cart.AddItem(productID, shopID, 2, cartdomain.ItemOptions{"size": "small", "color": "blue"})
	_ = cart.AddItem(productID, shopID, 3, cartdomain.ItemOptions{"size": "large", "color": "white"})
	item1ID := cart.Items[0].ID

	// Total across all styles
	total := cart.TotalProductQuantity(productID, shopID)
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}

	// Total excluding item 1
	totalExcl := cart.TotalProductQuantity(productID, shopID, item1ID)
	if totalExcl != 3 {
		t.Errorf("expected total excluding item1 to be 3, got %d", totalExcl)
	}
}
