package domain

import "errors"

var (
	ErrInvalidRating      = errors.New("rating must be between 1 and 5")
	ErrReviewNotFound     = errors.New("review not found")
	ErrDuplicateReview    = errors.New("review already submitted for this purchase")
	ErrUnverifiedPurchase = errors.New("Customer has not purchased this product")
	ErrForbiddenReviewOp  = errors.New("forbidden: cannot perform action on this review")
)
