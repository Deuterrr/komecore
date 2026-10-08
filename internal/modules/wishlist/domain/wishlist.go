package domain

import (
	"time"

	"github.com/google/uuid"
)

type WishlistItem struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
	CreatedAt  time.Time
}

type WishlistProductView struct {
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
