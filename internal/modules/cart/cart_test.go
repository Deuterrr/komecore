package cart_test

import (
	"testing"

	"komecore/internal/modules/cart"

	"github.com/google/uuid"
)

func TestCart_UpdateItemByID_QuantityOnly(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	opt := cart.ItemOptions{"size": "medium", "color": "black"}

	err := c.AddItem(productID, shopID, 2, opt)
	if err != nil {
		t.Fatalf("unexpected error adding item: %v", err)
	}

	targetID := c.Items[0].ID

	// Update quantity only with same options
	err = c.UpdateItemByID(targetID, 5, opt)
	if err != nil {
		t.Fatalf("unexpected error updating item: %v", err)
	}

	if len(c.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(c.Items))
	}
	if c.Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", c.Items[0].Quantity)
	}
	if !c.Items[0].ItemOptions.Equals(opt) {
		t.Errorf("expected options %+v, got %+v", opt, c.Items[0].ItemOptions)
	}
}

func TestCart_UpdateItemByID_OptionsChange_NoCollision(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	optOld := cart.ItemOptions{"size": "small", "color": "blue"}
	optNew := cart.ItemOptions{"size": "large", "color": "red"}

	_ = c.AddItem(productID, shopID, 1, optOld)
	targetID := c.Items[0].ID

	err := c.UpdateItemByID(targetID, 3, optNew)
	if err != nil {
		t.Fatalf("unexpected error updating item options: %v", err)
	}

	if len(c.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(c.Items))
	}
	if c.Items[0].Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", c.Items[0].Quantity)
	}
	if !c.Items[0].ItemOptions.Equals(optNew) {
		t.Errorf("expected options %+v, got %+v", optNew, c.Items[0].ItemOptions)
	}
}

func TestCart_UpdateItemByID_OptionsChange_WithCollision(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	productID := uuid.New()
	shopID := uuid.New()
	optA := cart.ItemOptions{"size": "small", "color": "blue"}
	optB := cart.ItemOptions{"size": "large", "color": "red"}

	_ = c.AddItem(productID, shopID, 2, optA)
	itemAID := c.Items[0].ID

	_ = c.AddItem(productID, shopID, 3, optB)
	itemBID := c.Items[1].ID

	// Change item A's options to optB (collision with item B)
	err := c.UpdateItemByID(itemAID, 2, optB)
	if err != nil {
		t.Fatalf("unexpected error updating with collision: %v", err)
	}

	// Item A should be soft-deleted, Item B should have qty 2 + 3 = 5
	var itemA, itemB *cart.CartItem
	for i := range c.Items {
		if c.Items[i].ID == itemAID {
			itemA = &c.Items[i]
		}
		if c.Items[i].ID == itemBID {
			itemB = &c.Items[i]
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
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}

	err := c.UpdateItemByID(uuid.New(), 1, cart.ItemOptions{})
	if err != cart.ErrCartItemNotFound {
		t.Errorf("expected ErrCartItemNotFound, got %v", err)
	}
}

func TestCart_UpdateItemByID_InvalidQuantity(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	pid := uuid.New()
	_ = c.AddItem(pid, uuid.New(), 1)
	targetID := c.Items[0].ID

	err := c.UpdateItemByID(targetID, 0)
	if err != cart.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for 0, got %v", err)
	}

	err = c.UpdateItemByID(targetID, -1)
	if err != cart.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity for -1, got %v", err)
	}
}

func TestCart_SetItem_NoOptions_UpdatesExistingItemInPlace(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	customOpt := cart.ItemOptions{"size": "large", "color": "white"}

	// Add item with non-default options
	_ = c.AddItem(productID, shopID, 2, customOpt)
	targetID := c.Items[0].ID

	// SetItem without options (as happens when PUT /carts/items/{shopID}/{productID} is called)
	err := c.SetItem(productID, shopID, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should update existing item in-place without duplicating
	if len(c.Items) != 1 {
		t.Fatalf("expected 1 item, got %d (item was duplicated)", len(c.Items))
	}
	if c.Items[0].ID != targetID {
		t.Errorf("expected item ID to remain %v, got %v", targetID, c.Items[0].ID)
	}
	if c.Items[0].Quantity != 7 {
		t.Errorf("expected quantity 7, got %d", c.Items[0].Quantity)
	}
	// ItemOptions should be preserved
	if !c.Items[0].ItemOptions.Equals(customOpt) {
		t.Errorf("expected options %+v to be preserved, got %+v", customOpt, c.Items[0].ItemOptions)
	}
}

func TestCart_RemoveItemByID(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()

	_ = c.AddItem(productID, shopID, 1, cart.ItemOptions{"size": "small", "color": "blue"})
	_ = c.AddItem(productID, shopID, 2, cart.ItemOptions{"size": "large", "color": "white"})
	item1ID := c.Items[0].ID
	item2ID := c.Items[1].ID

	// Remove item 2 specifically by its ID
	removed := c.RemoveItemByID(item2ID)
	if !removed {
		t.Fatalf("expected RemoveItemByID to return true")
	}

	if c.Items[0].DeletedAt != nil {
		t.Errorf("expected item 1 to remain active (not deleted)")
	}
	if c.Items[1].DeletedAt == nil {
		t.Errorf("expected item 2 to be soft-deleted")
	}

	// Remove item 1
	removed = c.RemoveItemByID(item1ID)
	if !removed {
		t.Fatalf("expected RemoveItemByID to return true for item 1")
	}
	if c.Items[0].DeletedAt == nil {
		t.Errorf("expected item 1 to be soft-deleted")
	}

	// Non-existent ID returns false
	if c.RemoveItemByID(uuid.New()) {
		t.Errorf("expected RemoveItemByID to return false for non-existent item")
	}
}

func TestCart_RemoveItem_WithOptions(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	optSmall := cart.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cart.ItemOptions{"size": "large", "color": "white"}

	_ = c.AddItem(productID, shopID, 1, optSmall)
	_ = c.AddItem(productID, shopID, 2, optLarge)

	// Remove item matching optLarge specifically
	removed := c.RemoveItem(productID, shopID, optLarge)
	if !removed {
		t.Fatalf("expected RemoveItem with options to return true")
	}

	if c.Items[0].DeletedAt != nil {
		t.Errorf("expected first item (optSmall) to remain active")
	}
	if c.Items[1].DeletedAt == nil {
		t.Errorf("expected second item (optLarge) to be deleted")
	}
}

func TestCart_SameProduct_DifferentStyles_CoexistAndModify(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()
	optSmall := cart.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cart.ItemOptions{"size": "large", "color": "red"}

	_ = c.AddItem(productID, shopID, 2, optSmall)
	_ = c.AddItem(productID, shopID, 3, optLarge)

	if len(c.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(c.Items))
	}
	smallItemID := c.Items[0].ID
	largeItemID := c.Items[1].ID

	// Update large item quantity only (no options passed)
	err := c.UpdateItemByID(largeItemID, 5)
	if err != nil {
		t.Fatalf("unexpected error updating large item: %v", err)
	}

	// Ensure large item updated in place and small item is untouched
	if len(c.Items) != 2 {
		t.Fatalf("expected still 2 items, got %d", len(c.Items))
	}
	if c.Items[0].ID != smallItemID || c.Items[0].Quantity != 2 || !c.Items[0].ItemOptions.Equals(optSmall) {
		t.Errorf("small item was unexpectedly modified: %+v", c.Items[0])
	}
	if c.Items[1].ID != largeItemID || c.Items[1].Quantity != 5 || !c.Items[1].ItemOptions.Equals(optLarge) {
		t.Errorf("large item options or quantity unexpected: %+v", c.Items[1])
	}
}

func TestCart_ChangeItemShop_DifferentStyles_DoNotMerge(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shop1 := uuid.New()
	shop2 := uuid.New()

	optSmall := cart.ItemOptions{"size": "small", "color": "blue"}
	optLarge := cart.ItemOptions{"size": "large", "color": "red"}

	// Shop 2 has Small
	_ = c.AddItem(productID, shop2, 2, optSmall)
	// Shop 1 has Large
	_ = c.AddItem(productID, shop1, 3, optLarge)
	largeItemID := c.Items[1].ID

	// Move large item from Shop 1 to Shop 2
	ok := c.ChangeItemShop(largeItemID, shop2)
	if !ok {
		t.Fatalf("expected ChangeItemShop to succeed")
	}

	// Both items should be active in Shop 2 as separate items (not merged into Small)
	if c.Items[0].DeletedAt != nil {
		t.Errorf("expected Small item to remain active")
	}
	if c.Items[1].DeletedAt != nil {
		t.Errorf("expected Large item to remain active (not soft-deleted)")
	}
	if c.Items[1].ShopID != shop2 {
		t.Errorf("expected Large item shop_id to be shop2")
	}
	if c.Items[0].Quantity != 2 || c.Items[1].Quantity != 3 {
		t.Errorf("expected quantities 2 and 3, got %d and %d", c.Items[0].Quantity, c.Items[1].Quantity)
	}
}

func TestCart_ChangeItemShop_SameStyle_Merges(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shop1 := uuid.New()
	shop2 := uuid.New()

	optLarge := cart.ItemOptions{"size": "large", "color": "red"}

	// Shop 2 has Large (qty 2)
	_ = c.AddItem(productID, shop2, 2, optLarge)
	// Shop 1 has Large (qty 3)
	_ = c.AddItem(productID, shop1, 3, optLarge)
	shop1LargeItemID := c.Items[1].ID

	// Move Large item from Shop 1 to Shop 2 (same product and same style)
	ok := c.ChangeItemShop(shop1LargeItemID, shop2)
	if !ok {
		t.Fatalf("expected ChangeItemShop to succeed")
	}

	// Should merge into Shop 2 Large item (qty 5) and soft-delete Shop 1 item
	if c.Items[0].Quantity != 5 {
		t.Errorf("expected merged quantity 5, got %d", c.Items[0].Quantity)
	}
	if c.Items[1].DeletedAt == nil {
		t.Errorf("expected moved item to be soft-deleted after merge")
	}
}

func TestCart_TotalProductQuantity(t *testing.T) {
	c := &cart.Cart{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
	}
	productID := uuid.New()
	shopID := uuid.New()

	_ = c.AddItem(productID, shopID, 2, cart.ItemOptions{"size": "small", "color": "blue"})
	_ = c.AddItem(productID, shopID, 3, cart.ItemOptions{"size": "large", "color": "white"})
	item1ID := c.Items[0].ID

	// Total across all styles
	total := c.TotalProductQuantity(productID, shopID)
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}

	// Total excluding item 1
	totalExcl := c.TotalProductQuantity(productID, shopID, item1ID)
	if totalExcl != 3 {
		t.Errorf("expected total excluding item1 to be 3, got %d", totalExcl)
	}
}
