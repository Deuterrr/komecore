package order

import (
	"github.com/google/uuid"

	"komecore/internal/modules/cart"
)

type PricingItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cart.ItemOptions
	Quantity    int
}

type PricingShopInput struct {
	ShopID         uuid.UUID
	CourierCode    *string
	CourierService *string
	Items          []PricingItemInput
}

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

type CourierOption struct {
	Code    string
	Service string
	Name    string
	ETD     string
	Fee     int64
}

type SelectedCourierResult struct {
	Code    string
	Service string
	Fee     int64
}

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
