package apperror

const (
	// Generic / Fallback machine-readable error codes
	CodeBadRequest          = "BAD_REQUEST"
	CodeInvalidInput        = "INVALID_INPUT"
	CodeNotFound            = "NOT_FOUND"
	CodeUnauthorized        = "AUTH_UNAUTHORIZED"
	CodeForbidden           = "AUTH_FORBIDDEN"
	CodeConflict            = "CONFLICT"
	CodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	CodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"

	// Auth domain codes
	CodeAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	CodeAuthTokenExpired       = "AUTH_TOKEN_EXPIRED"
	CodeAuthUnauthorized       = "AUTH_UNAUTHORIZED"
	CodeAuthForbidden          = "AUTH_FORBIDDEN"

	// Catalog domain codes
	CodeProductNotFound = "PRODUCT_NOT_FOUND"

	// Inventory domain codes
	CodeInsufficientStock = "INSUFFICIENT_STOCK"

	// Cart domain codes
	CodeCartItemNotFound = "CART_ITEM_NOT_FOUND"

	// Coupon domain codes
	CodeCouponNotFound       = "COUPON_NOT_FOUND"
	CodeCouponExpired        = "COUPON_EXPIRED"
	CodeCouponMinSpendNotMet = "COUPON_MIN_SPEND_NOT_MET"
	CodeCouponQuotaExhausted = "COUPON_QUOTA_EXHAUSTED"

	// Order domain codes
	CodeOrderNotFound     = "ORDER_NOT_FOUND"
	CodeOrderStateInvalid = "ORDER_STATE_INVALID"

	// Payment domain codes
	CodePaymentExpired = "PAYMENT_EXPIRED"

	// Review domain codes
	CodeUnverifiedPurchase = "UNVERIFIED_PURCHASE"
	CodeDuplicateReview    = "DUPLICATE_REVIEW"
)

// DefaultCodeForType returns the fallback machine-readable code for a given ErrorType.
func DefaultCodeForType(t ErrorType) string {
	switch t {
	case ErrTypeBadRequest:
		return CodeBadRequest
	case ErrTypeInvalidInput:
		return CodeInvalidInput
	case ErrTypeNotFound:
		return CodeNotFound
	case ErrTypeUnauthorized:
		return CodeAuthUnauthorized
	case ErrTypeForbidden:
		return CodeAuthForbidden
	case ErrTypeConflict:
		return CodeConflict
	case ErrTypeMethodNotAllowed:
		return CodeMethodNotAllowed
	case ErrTypeTooManyRequests:
		return CodeRateLimitExceeded
	case ErrTypeInternal:
		return CodeInternalServerError
	default:
		return CodeInternalServerError
	}
}
