package domain

import "errors"

var (
	ErrInvalidStock         = errors.New("stock cannot be negative")
	ErrInvalidReserved      = errors.New("reserved stock cannot be negative")
	ErrReservedExceedsStock = errors.New("reserved stock cannot exceed stock")

	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrInsufficientReserved = errors.New("insufficient reserved stock")
)
