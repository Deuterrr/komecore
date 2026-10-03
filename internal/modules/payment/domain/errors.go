package domain

import "errors"

var ErrInvalidName = errors.New("name is required")
var ErrInvalidCode = errors.New("code is required")
var ErrInvalidProvider = errors.New("provider is required")
var ErrInvalidType = errors.New("invalid type")
var ErrInvalidFeeType = errors.New("invalid fee type")
var ErrInvalidFeeFixed = errors.New("fixed fee must be greater than 0")
var ErrInvalidFeePercentage = errors.New("percentage must be between 0 and 1")
var ErrInvalidFeeMax = errors.New("maximum fee must be greater than or equal to 0")

var ErrPaymentAccountConflict = errors.New("payment account already exists")

var ErrPaymentAccountNotFound = errors.New("payment account not found")

var ErrInvalidAccountNumber = errors.New("account number is required")
var ErrInvalidAccountName = errors.New("account name is required")
var ErrInvalidPhoneNumber = errors.New("phone number is required")
var ErrInvalidQRString = errors.New("qr string is required")

var ErrPaymentMethodConflict = errors.New("payment method already exists")

var ErrPaymentMethodNotFound = errors.New("payment method not found")

var ErrUnsupportedPaymentMethod = errors.New("unsupported payment method type")
var ErrNoActivePaymentAccount = errors.New("no active payment account")
