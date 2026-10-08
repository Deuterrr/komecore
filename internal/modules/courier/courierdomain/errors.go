package courierdomain

import "errors"

var (
	ErrShopNotFound = errors.New("shop not found")

	ErrCourierNotFound         = errors.New("courier not found or unsupported")
	ErrCourierNameRequired     = errors.New("courier name is required when activating courier")
	ErrCourierLocationRequired = errors.New("location address is required when activating courier")
	ErrCourierNotPending       = errors.New("courier is not in pending verification status")
	ErrCourierAlreadyActive    = errors.New("courier is already active and verified")

	ErrInvalidVerificationAction = errors.New("invalid verification action: must be 'verify' or 'reject'")

	ErrInvalidCoordinates = errors.New("Invalid location coordinates. Latitude must be between -90 and 90, and longitude between -180 and 180")
)
