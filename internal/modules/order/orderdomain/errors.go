package orderdomain

import "errors"

var (
	ErrMissingSLAFields = errors.New("orders in confirmed or processing status must have confirmed_at and handling_expires_at populated")

	ErrOrderNotFound   = errors.New("order not found")
	ErrOrderHasNoItems = errors.New("order has no items")

	ErrCustomerAddressNotFound = errors.New("customer address not found")

	ErrShopAddressNotFound = errors.New("shop address not found")

	ErrPaymentNotConfirmed = errors.New("cannot confirm order without confirmed payment")
	ErrPaymentNotPaid      = errors.New("cannot move order to processing without confirmed payment")

	ErrEmptyShipmentItems = errors.New("each shipment must contain at least one order item")
	ErrMissingCourierInfo = errors.New("shipment is missing courier information")
)
