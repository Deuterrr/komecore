package order

import (
	"komecore/internal/modules/cart"

	"github.com/google/uuid"
)

// OrderItem represents an immutable snapshot of an individual purchased item at checkout time.
// It preserves price, merchant shop provenance, and variant options even if underlying catalog products mutate.
type OrderItem struct {
	ID      uuid.UUID
	OrderID uuid.UUID
	// ShipmentID links the item to a fulfilled package; nil while the item remains unshipped in processing.
	ShipmentID *uuid.UUID

	ShopID   uuid.UUID
	ShopName string

	ProductID   uuid.UUID
	ProductName string

	Quantity  int
	UnitPrice int64
	Subtotal  int64

	// ItemOptions captures variant specifications (e.g. size, color) snapshotted from the cart.
	ItemOptions cart.ItemOptions

	// CourierCode records the selected shipping provider for the merchant shop owning this item.
	CourierCode *string
	// CourierService records the selected service level (e.g. REG, YES) for the shop owning this item.
	CourierService *string
	// ShippingFee captures the shop-level delivery fee snapshot assigned at checkout.
	// It is recorded per line item for historical auditability and is not aggregated per piece.
	ShippingFee int64
}
