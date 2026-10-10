package orderhttp

import (
	"github.com/google/uuid"
)

type checkoutItemRequest struct {
	ProductID   *string           `json:"product_id,omitempty"`
	CartItemID  *string           `json:"cart_item_id,omitempty"`
	Quantity    int               `json:"quantity"`
	ItemOptions map[string]string `json:"item_options,omitempty"`
}

type selectedCourierRequest struct {
	Code    string `json:"code"`
	Service string `json:"service"`
}

type checkoutShopRequest struct {
	ShopID  string                  `json:"shop_id"`
	Items   []checkoutItemRequest   `json:"items"`
	Courier *selectedCourierRequest `json:"courier"`
}

type checkoutRequest struct {
	CouponCode *string               `json:"coupon_code,omitempty"`
	Shops      []checkoutShopRequest `json:"shops"`
}

type checkoutCalculateRequest struct {
	PaymentMethodID *string               `json:"payment_method_id"`
	AddressID       *string               `json:"address_id"`
	CouponCode      *string               `json:"coupon_code,omitempty"`
	Shops           []checkoutShopRequest `json:"shops"`
}

type checkoutAddressResponse struct {
	ID            uuid.UUID `json:"id"`
	RecipientName string    `json:"recipient_name"`
	Phone         *string   `json:"phone"`
	FullAddress   string    `json:"full_address"`
}

type checkoutCouriersResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Service string `json:"service"`
	ETD     string `json:"etd"`
	Fee     int64  `json:"fee"`
}

type checkoutItemResponse struct {
	ProductID   uuid.UUID         `json:"product_id"`
	CartItemID  *uuid.UUID        `json:"cart_item_id,omitempty"`
	ShopID      uuid.UUID         `json:"shop_id"`
	Name        string            `json:"name"`
	Price       int64             `json:"price"`
	Quantity    int               `json:"quantity"`
	Subtotal    int64             `json:"subtotal"`
	ItemOptions map[string]string `json:"item_options,omitempty"`
}

type selectedCourierResponse struct {
	Code    string `json:"code"`
	Service string `json:"service"`
	Fee     int64  `json:"fee"`
}

type shopResponse struct {
	ShopID       uuid.UUID                  `json:"shop_id"`
	ShopName     string                     `json:"name"`
	ShopSlug     string                     `json:"slug"`
	Subtotal     int64                      `json:"subtotal"`
	Total        *int64                     `json:"total"`
	Items        []checkoutItemResponse     `json:"items"`
	CostCouriers []checkoutCouriersResponse `json:"cost_couriers"`
}

type paymentMethodResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Fee         int64     `json:"fee"`
	Subtotal    int64     `json:"subtotal"`
	Total       int64     `json:"total"`
}

type checkoutResponse struct {
	Address        checkoutAddressResponse `json:"address"`
	Shops          []shopResponse          `json:"shops"`
	Subtotal       int64                   `json:"subtotal"`
	TotalShipping  int64                   `json:"total_shipping"`
	DiscountAmount int64                   `json:"discount_amount"`
	CouponCode     *string                 `json:"coupon_code,omitempty"`
	TotalAll       *int64                  `json:"total"`
	PaymentMethods []paymentMethodResponse `json:"payment_methods"`
}

type shopCalculateResponse struct {
	ShopID          uuid.UUID               `json:"shop_id"`
	ShopName        string                  `json:"name"`
	ShopSlug        string                  `json:"slug"`
	Subtotal        int64                   `json:"subtotal"`
	Total           *int64                  `json:"total"`
	SelectedCourier selectedCourierResponse `json:"selected_courier"`
	Items           []checkoutItemResponse  `json:"items"`
}

type checkoutCalculateResponse struct {
	Address                checkoutAddressResponse `json:"address"`
	Shops                  []shopCalculateResponse `json:"shops"`
	Subtotal               int64                   `json:"subtotal"`
	TotalShipping          int64                   `json:"total_shipping"`
	DiscountAmount         int64                   `json:"discount_amount"`
	CouponCode             *string                 `json:"coupon_code,omitempty"`
	TotalAll               *int64                  `json:"total"`
	SelectedPaymentMethods paymentMethodResponse   `json:"selected_payment_method"`
}
