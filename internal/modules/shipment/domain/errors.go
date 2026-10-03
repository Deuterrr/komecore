package domain

import "errors"

var (
	ErrInvalidCourier     = errors.New("courier is invalid")
	ErrInvalidService     = errors.New("service is invalid")
	ErrInvalidOrigin      = errors.New("origin is invalid")
	ErrInvalidDestination = errors.New("destination is invalid")

	ErrInvalidCost   = errors.New("cost must be greater than or equal to 0")
	ErrInvalidWeight = errors.New("weight must be greater than 0")

	ErrInvalidStatus            = errors.New("invalid status")
	ErrInvalidFulfillmentMethod = errors.New("fulfillment method is invalid")
)

var (
	ErrInvalidEventStatus      = ErrInvalidStatus
	ErrInvalidEventDescription = errors.New("event description is invalid")
)

var (
	ErrInvalidDisplayName = errors.New("display name is invalid")
)

var (
	ErrInvalidRoute       = errors.New("origin and destination cannot be the same")
	ErrNoCourierSelected  = errors.New("at least one courier must be selected")
	ErrUnsupportedCourier = errors.New("unsupported courier selected")
)
