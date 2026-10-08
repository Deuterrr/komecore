package productdomain

import (
	"time"

	"github.com/google/uuid"
)

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	ID          uuid.UUID
	SKU         string
	Name        string
	Slug        string
	Description *string
	Status      ProductStatus

	Price  int64
	Weight *float64

	AverageRating float64
	ReviewCount   int

	CreatedAt  time.Time
	UpdatedAt  *time.Time
	ArchivedAt *time.Time
}

type ProductWithInventory struct {
	Product       Product
	TotalStock    int
	ReservedStock int
}

func (p *Product) Validate() error {
	if p.Name == "" {
		return ErrInvalidProductName
	}

	if p.Slug == "" {
		return ErrInvalidSlug
	}

	if p.Price <= 0 {
		return ErrInvalidProductPrice
	}

	return nil
}
