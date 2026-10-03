package domain

import "errors"

var (
	ErrCannotDeleteDefaultAddress = errors.New("default address cannot be deleted")
	ErrNotFoundDefaultAddress     = errors.New("default address not found")
	ErrAddressNotFound            = errors.New("address not found")
	ErrAddressLimitReached        = errors.New("maximum address limit reached")
	ErrInvalidCoordinates         = errors.New("invalid coordinates: latitude must be between -90 and 90, longitude between -180 and 180")
)
