package carthttp

import (
	"github.com/google/uuid"
)

type addItemRequest struct {
	ProductID   string            `json:"product_id"`
	ShopID      string            `json:"shop_id"`
	Quantity    int               `json:"quantity"`
	ItemOptions map[string]string `json:"item_options,omitempty"`
}

type updateItemRequest struct {
	Quantity    int               `json:"quantity"`
	ItemOptions map[string]string `json:"item_options,omitempty"`
}

type updateItemByIDRequest struct {
	Quantity    int               `json:"quantity"`
	ItemOptions map[string]string `json:"item_options,omitempty"`
}

type cartResponse struct {
	CartID uuid.UUID      `json:"cart_id"`
	Items  []cartItemView `json:"items"`
	Total  int64          `json:"total"`
}

type productImageResponse struct {
	Thumbnail *string `json:"thumbnail,omitempty"`
	Preview   *string `json:"preview,omitempty"`
	Detail    *string `json:"detail,omitempty"`
}

type cartItemView struct {
	CartItemID  uuid.UUID            `json:"cart_item_id"`
	ProductID   uuid.UUID            `json:"product_id"`
	ShopID      uuid.UUID            `json:"shop_id"`
	ShopName    string               `json:"shop_name,omitempty"`
	ShopSlug    string               `json:"shop_slug,omitempty"`
	Name        string               `json:"name"`
	Price       int64                `json:"price"`
	Subtotal    int64                `json:"subtotal"`
	Quantity    int                  `json:"quantity"`
	Image       productImageResponse `json:"images"`
	ItemOptions map[string]string    `json:"item_options,omitempty"`
}
