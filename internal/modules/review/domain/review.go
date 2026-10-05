package domain

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	CustomerID uuid.UUID
	OrderID   uuid.UUID
	Rating    int
	Title     *string
	Comment   *string

	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (r *Review) Validate() error {
	if r.Rating < 1 || r.Rating > 5 {
		return ErrInvalidRating
	}

	if r.ProductID == uuid.Nil {
		return errors.New("invalid product id")
	}

	if r.CustomerID == uuid.Nil {
		return errors.New("invalid customer id")
	}

	if r.OrderID == uuid.Nil {
		return errors.New("invalid order id")
	}

	if r.Title != nil {
		trimmed := strings.TrimSpace(*r.Title)
		if len(trimmed) > 120 {
			return errors.New("title cannot exceed 120 characters")
		}
	}

	return nil
}

type ReviewWithCustomer struct {
	Review       Review
	CustomerName string
	AvatarURL    *string
}

type ProductRatingSummary struct {
	AverageRating float64
	ReviewCount   int
}

func NewProductRatingSummary(rawAvg float64, count int) ProductRatingSummary {
	return ProductRatingSummary{
		AverageRating: math.Round(rawAvg*100) / 100,
		ReviewCount:   count,
	}
}
