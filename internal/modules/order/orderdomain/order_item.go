package orderdomain

import (
	"komecore/internal/modules/cart/cartdomain"

	"github.com/google/uuid"
)

type OrderItem struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	ShipmentID *uuid.UUID

	ShopID   uuid.UUID
	ShopName string

	ProductID   uuid.UUID
	ProductName string

	Quantity  int
	UnitPrice int64
	Subtotal  int64

	// ItemOptions captures generic variant specifications stored as JSONB
	ItemOptions cartdomain.ItemOptions

	// CourierCode is the selected courier code
	// for the shop that owns this item
	CourierCode *string

	// CourierService is the selected courier service
	// for the shop that owns this item
	CourierService *string

	// ShippingFee is the total shipping fee charged for
	// the shop that owns this item
	//
	// It represents the shop-level shipping cost captured
	// at checkout time and is not allocated
	// or calculated on a per-item basis
	ShippingFee int64
}
