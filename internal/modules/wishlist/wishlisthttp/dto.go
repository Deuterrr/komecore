package wishlisthttp

import (
	"komecore/internal/modules/wishlist"
)

type wishlistResponse struct {
	Items []wishlist.WishlistProductView `json:"items"`
	Total int                            `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}
