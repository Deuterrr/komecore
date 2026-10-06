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
	ProductID    uuid.UUID
	SKU          string
	Name         string
	Slug         string
	Price        int64
	PrimaryImage *string
	InStock      bool
	TotalStock   int
	CreatedAt    time.Time
}
