package http

import (
	"time"

	"github.com/google/uuid"
)

type wishlistItemResponse struct {
	ProductID    uuid.UUID `json:"product_id"`
	SKU          string    `json:"sku"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Price        int64     `json:"price"`
	PrimaryImage *string   `json:"primary_image"`
	InStock      bool      `json:"in_stock"`
	TotalStock   int       `json:"total_stock"`
	CreatedAt    time.Time `json:"created_at"`
}

type wishlistResponse struct {
	Items []wishlistItemResponse `json:"items"`
	Total int                    `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}
