package persistence

import (
	"time"

	"github.com/google/uuid"
)

type productModel struct {
	ID          uuid.UUID `db:"id"`
	SKU         string    `db:"sku"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	Status      string    `db:"status"`

	BasePrice string  `db:"base_price"`
	Weight    *string `db:"weight"`

	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  *time.Time `db:"updated_at"`
	ArchivedAt *time.Time `db:"archived_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type productImageRow struct {
	ID        uuid.UUID
	ProductID uuid.UUID

	ThumbURL string
	ThumbKey string

	PreviewURL string
	PreviewKey string

	DetailURL string
	DetailKey string

	IsPrimary    bool
	DisplayOrder int
	CreatedAt    time.Time
}
