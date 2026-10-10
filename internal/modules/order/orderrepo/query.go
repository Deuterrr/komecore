package orderrepo

import (
	"time"

	"komecore/internal/modules/cart/cartdomain"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type PricingItemInput struct {
	ProductID   uuid.UUID
	CartItemID  *uuid.UUID
	ItemOptions cartdomain.ItemOptions
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
	ItemOptions cartdomain.ItemOptions
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

var (
	OrderSortLatest pagination.SortKey = "latest"
	OrderSortNumber pagination.SortKey = "number"
	OrderSortTotal  pagination.SortKey = "total"
	OrderSortStatus pagination.SortKey = "status"
	OrderSortModify pagination.SortKey = "modify"
)

type FindOrderParams struct {
	ID         *uuid.UUID
	Number     *string
	CustomerID *uuid.UUID
	ShopID     *uuid.UUID
	ShopIDs    []uuid.UUID
	Status     *string
	Statuses   []string
	FromDate   *time.Time
	ToDate     *time.Time

	Pagination pagination.Pagination
	Sorts      pagination.Sorts
}
