package cartdomain

import (
	"time"

	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type Cart struct {
	ID uuid.UUID

	CustomerID uuid.UUID

	Items []CartItem

	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (c *Cart) AddItem(productID uuid.UUID, shopID uuid.UUID, qty int, options ...ItemOptions) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}

	var opt ItemOptions
	if len(options) > 0 {
		opt = options[0].Normalized()
	} else {
		opt = ItemOptions{}.Normalized()
	}

	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID && item.ItemOptions.Equals(opt) {
			item.Quantity += qty
			return nil
		}
	}

	c.Items = append(c.Items, CartItem{
		ID:          uuid.New(),
		ProductID:   productID,
		ShopID:      shopID,
		Quantity:    qty,
		ItemOptions: opt,
	})

	return nil
}

func (c *Cart) SetItem(productID uuid.UUID, shopID uuid.UUID, qty int, options ...ItemOptions) error {
	if qty < 0 {
		return ErrInvalidQuantity
	}

	if len(options) == 0 {
		// When options are omitted, match any existing item for this product & shop
		for i := range c.Items {
			item := &c.Items[i]
			if item.DeletedAt != nil {
				continue
			}
			if item.ProductID == productID && item.ShopID == shopID {
				if qty == 0 {
					now := appclock.Now()
					item.DeletedAt = &now
					return nil
				}
				item.Quantity = qty
				return nil
			}
		}

		if qty == 0 {
			return nil
		}

		opt := ItemOptions{}.Normalized()
		c.Items = append(c.Items, CartItem{
			ID:          uuid.New(),
			ProductID:   productID,
			ShopID:      shopID,
			Quantity:    qty,
			ItemOptions: opt,
			DeletedAt:   nil,
		})
		return nil
	}

	opt := options[0].Normalized()

	for i := range c.Items {
		item := &c.Items[i]
		if item.ProductID == productID && item.ShopID == shopID && item.ItemOptions.Equals(opt) {
			if qty == 0 {
				now := appclock.Now()
				item.DeletedAt = &now
				return nil
			}

			item.Quantity = qty
			item.DeletedAt = nil
			return nil
		}
	}

	if qty == 0 {
		return nil
	}

	c.Items = append(c.Items, CartItem{
		ID:          uuid.New(),
		ProductID:   productID,
		ShopID:      shopID,
		Quantity:    qty,
		ItemOptions: opt,
		DeletedAt:   nil,
	})

	return nil
}

// UpdateItemByID updates the quantity and/or ItemOptions of the cart item
// identified by cartItemID. If newOptions differs from the current item's
// options, and another item with the same productID+shopID+newOptions already
// exists, the two items are merged (quantities summed) and the original is
// soft-deleted.
// Returns ErrCartItemNotFound if no active item matches cartItemID.
func (c *Cart) UpdateItemByID(cartItemID uuid.UUID, qty int, options ...ItemOptions) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}

	var target *CartItem
	for i := range c.Items {
		item := &c.Items[i]
		if item.ID == cartItemID && item.DeletedAt == nil {
			target = item
			break
		}
	}
	if target == nil {
		return ErrCartItemNotFound
	}

	var opt ItemOptions
	if len(options) > 0 {
		opt = options[0].Normalized()
	} else {
		opt = target.ItemOptions.Normalized()
	}

	// If options haven't changed, simply update quantity
	if target.ItemOptions.Equals(opt) {
		target.Quantity = qty
		return nil
	}

	// Options changed — check if another active item with the new options already exists
	productID := target.ProductID
	shopID := target.ShopID
	for i := range c.Items {
		item := &c.Items[i]
		if item.ID == cartItemID || item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID && item.ItemOptions.Equals(opt) {
			// Merge: add quantity to existing item, soft-delete the target item
			item.Quantity += qty
			now := appclock.Now()
			target.DeletedAt = &now
			return nil
		}
	}

	// No collision — update options and quantity in place
	target.ItemOptions = opt
	target.Quantity = qty
	return nil
}

func (c *Cart) RemoveItem(productID uuid.UUID, shopID uuid.UUID, options ...ItemOptions) bool {
	var opt *ItemOptions
	if len(options) > 0 {
		normalized := options[0].Normalized()
		opt = &normalized
	}

	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID {
			if opt == nil || item.ItemOptions.Equals(*opt) {
				now := appclock.Now()
				item.DeletedAt = &now
				return true
			}
		}
	}
	return false
}

// RemoveItemByID soft-deletes any cart item by its own UUID.
// Returns false if the item was not found or already deleted.
func (c *Cart) RemoveItemByID(cartItemID uuid.UUID) bool {
	for i := range c.Items {
		if c.Items[i].ID == cartItemID && c.Items[i].DeletedAt == nil {
			now := appclock.Now()
			c.Items[i].DeletedAt = &now
			return true
		}
	}
	return false
}

// RemoveProduct soft-deletes an active cart item matching productID regardless of shopID.
// Returns false if the item was not found or already deleted.
func (c *Cart) RemoveProduct(productID uuid.UUID) bool {
	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID {
			now := appclock.Now()
			item.DeletedAt = &now
			return true
		}
	}
	return false
}

func (c *Cart) HasItem(productID uuid.UUID, shopID uuid.UUID, options ...ItemOptions) bool {
	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID {
			if len(options) == 0 || item.ItemOptions.Equals(options[0]) {
				return true
			}
		}
	}
	return false
}

func (c *Cart) FindItem(productID uuid.UUID, shopID uuid.UUID, options ...ItemOptions) *CartItem {
	var opt *ItemOptions
	if len(options) > 0 {
		normalized := options[0].Normalized()
		opt = &normalized
	}

	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID {
			if opt == nil || item.ItemOptions.Equals(*opt) {
				return item
			}
		}
	}

	return nil
}

func (c *Cart) HasProductInAnotherShop(productID uuid.UUID, shopID uuid.UUID) bool {
	for _, item := range c.Items {
		if item.DeletedAt != nil {
			continue
		}
		if item.ProductID == productID && item.ShopID != shopID {
			return true
		}
	}

	return false
}

// ChangeItemShop updates the fulfillment shop for a cart item identified by cartItemID.
// If an item with the same product ID already exists in newShopID,
// their quantities are merged and the original item is soft-deleted.
func (c *Cart) ChangeItemShop(cartItemID uuid.UUID, newShopID uuid.UUID) bool {
	var targetIdx = -1
	for i := range c.Items {
		if c.Items[i].ID == cartItemID && c.Items[i].DeletedAt == nil {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return false
	}

	item := &c.Items[targetIdx]
	if item.ShopID == newShopID {
		return true
	}

	// Check if exact product + style already exists in newShopID
	productID := item.ProductID
	for i := range c.Items {
		if i != targetIdx && c.Items[i].DeletedAt == nil &&
			c.Items[i].ProductID == productID &&
			c.Items[i].ShopID == newShopID &&
			c.Items[i].ItemOptions.Equals(item.ItemOptions) {

			// Merge quantity into existing item for newShopID and soft-delete target item
			c.Items[i].Quantity += item.Quantity
			now := appclock.Now()
			item.DeletedAt = &now
			return true
		}
	}

	item.ShopID = newShopID
	return true
}

// TotalProductQuantity calculates the sum of quantities for all active items matching
// productID and shopID in the cart, optionally excluding a specific cart item by ID.
func (c *Cart) TotalProductQuantity(productID uuid.UUID, shopID uuid.UUID, excludeItemID ...uuid.UUID) int {
	var exclude uuid.UUID
	if len(excludeItemID) > 0 {
		exclude = excludeItemID[0]
	}

	total := 0
	for i := range c.Items {
		item := &c.Items[i]
		if item.DeletedAt != nil {
			continue
		}
		if item.ID == exclude {
			continue
		}
		if item.ProductID == productID && item.ShopID == shopID {
			total += item.Quantity
		}
	}
	return total
}
