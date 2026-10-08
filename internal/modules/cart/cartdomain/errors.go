package cartdomain

import "errors"

var (
	ErrInvalidShopID = errors.New("invalid shop id")

	ErrProductAlreadyAssignedToShop = errors.New("product already exists in cart from another shop")
	ErrProductNotFound              = errors.New("product not found")

	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientStock = errors.New("insufficient stock")

	ErrShopCouriersNotFound = errors.New("shop has no active courier service available")

	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")
)
