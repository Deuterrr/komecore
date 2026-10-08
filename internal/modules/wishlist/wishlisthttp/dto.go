package wishlisthttp

import (
	"komecore/internal/modules/wishlist/wishlistdomain"
)

type wishlistResponse struct {
	Items []wishlistdomain.WishlistProductView `json:"items"`
	Total int                          `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}
