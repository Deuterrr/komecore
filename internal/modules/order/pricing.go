package order

import (
	"github.com/google/uuid"

	"komecore/internal/modules/cart"
)

// PricingItemInput specifies a product item to be evaluated by the pricing engine.
// When CartItemID is provided, ProductID and ItemOptions are hydrated from persistent cart storage.
type PricingItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cart.ItemOptions
	Quantity    int
}

// PricingShopInput groups items by merchant shop to evaluate shop-specific shipping rates.
// CourierCode and CourierService default to the least expensive available rate when omitted.
type PricingShopInput struct {
	ShopID         uuid.UUID
	CourierCode    *string
	CourierService *string
	Items          []PricingItemInput
}

// PricingInput defines the customer basket parameters for price calculation.
// AddressID and PaymentMethodID are optional; when omitted, the customer's
// default address and all available payment methods are evaluated.
type PricingInput struct {
	CustomerID      uuid.UUID
	AddressID       *uuid.UUID
	PaymentMethodID *uuid.UUID
	CouponCode      *string
	Shops           []PricingShopInput
}

type PricingItemResult struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cart.ItemOptions
	ProductName string
	Quantity    int
	UnitPrice   int64
	Subtotal    int64
	WeightGrams int
}

// CourierOption contains live delivery quotes returned by the logistics provider,
// including courier code, service name, estimated days (ETD), and shipping fee.
type CourierOption struct {
	Code    string
	Service string
	Name    string
	ETD     string
	Fee     int64
}

// SelectedCourierResult represents the resolved delivery option for a merchant shop,
// holding either the customer's explicit choice or the lowest-cost fallback fee.
type SelectedCourierResult struct {
	Code    string
	Service string
	Fee     int64
}

// PricingShopResult captures calculated subtotals, selected shipping fees,
// and all available courier rate options for a single merchant shop.
type PricingShopResult struct {
	ShopID          uuid.UUID
	ShopName        string
	ShopSlug        string
	Items           []PricingItemResult
	SelectedCourier SelectedCourierResult
	CourierOptions  []CourierOption
	Subtotal        int64
	Total           int64
}

// PaymentMethodPricingResult calculates payment gateway surcharges (flat, percentage, or mixed)
// evaluated against the post-discount order total.
type PaymentMethodPricingResult struct {
	PaymentMethodID uuid.UUID
	Name            string
	Type            string
	Description     string
	Fee             int64
	Subtotal        int64
	Total           int64
}

type PricingAddressResult struct {
	ID            uuid.UUID
	RecipientName string
	Phone         *string
	FullAddress   string
}

// PricingResult represents the authoritative pricing breakdown for a checkout session.
// GrandTotal guarantees exact balance: Subtotal + TotalShippingFee + PaymentMethodFee - DiscountAmount.
type PricingResult struct {
	Address               PricingAddressResult
	Shops                 []PricingShopResult
	Subtotal              int64
	TotalShippingFee      int64
	DiscountAmount        int64
	CouponCode            *string
	GrandTotal            int64
	PaymentMethods        []PaymentMethodPricingResult
	SelectedPaymentMethod *PaymentMethodPricingResult
}
