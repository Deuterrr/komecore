package payment

import "errors"

// Input validation failures for payment method administration and pricing rules.
var (
	ErrInvalidName          = errors.New("name is required")
	ErrInvalidCode          = errors.New("code is required")
	ErrInvalidProvider      = errors.New("provider is required")
	ErrInvalidType          = errors.New("invalid type")
	ErrInvalidFeeType       = errors.New("invalid fee type")
	ErrInvalidFeeFixed      = errors.New("fixed fee must be greater than 0")
	ErrInvalidFeePercentage = errors.New("percentage must be between 0 and 1")
	ErrInvalidFeeMax        = errors.New("maximum fee must be greater than or equal to 0")
)

// Bank account and manual payout destination validation failures.
var (
	ErrPaymentAccountConflict = errors.New("payment account already exists")
	ErrPaymentAccountNotFound = errors.New("payment account not found")
	ErrInvalidAccountNumber   = errors.New("account number is required")
	ErrInvalidAccountName     = errors.New("account name is required")
	ErrInvalidPhoneNumber     = errors.New("phone number is required")
	ErrInvalidQRString        = errors.New("qr string is required")
)

// Relational conflicts and missing payment channel configurations.
var (
	ErrPaymentMethodConflict    = errors.New("payment method already exists")
	ErrPaymentMethodNotFound    = errors.New("payment method not found")
	ErrUnsupportedPaymentMethod = errors.New("unsupported payment method type")
	ErrNoActivePaymentAccount   = errors.New("no active payment account")
)
