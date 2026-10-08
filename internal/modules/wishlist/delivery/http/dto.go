package http

import (
	"komecore/internal/modules/wishlist/domain"
)

type wishlistResponse struct {
	Items []domain.WishlistProductView `json:"items"`
	Total int                          `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}
