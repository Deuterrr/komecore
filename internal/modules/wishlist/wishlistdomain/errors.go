package wishlistdomain

import "errors"

var (
	ErrWishlistItemAlreadyExists = errors.New("product is already in wishlist")
	ErrWishlistItemNotFound      = errors.New("product not found in wishlist")
	ErrProductNotFound           = errors.New("product not found")
	ErrInvalidProductID          = errors.New("invalid product id")
	ErrInvalidCustomerID         = errors.New("invalid customer id")
)
