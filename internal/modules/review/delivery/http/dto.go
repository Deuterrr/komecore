package http

import (
	"time"

	"github.com/google/uuid"
)

type createReviewRequest struct {
	Rating  int        `json:"rating"`
	Title   *string    `json:"title"`
	Comment *string    `json:"comment"`
	OrderID *uuid.UUID `json:"order_id"`
}

type reviewResponse struct {
	ID         uuid.UUID  `json:"id"`
	ProductID  uuid.UUID  `json:"product_id"`
	CustomerID uuid.UUID  `json:"customer_id"`
	OrderID    uuid.UUID  `json:"order_id"`
	Rating     int        `json:"rating"`
	Title      *string    `json:"title"`
	Comment    *string    `json:"comment"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

type reviewListItemResponse struct {
	ID           uuid.UUID  `json:"id"`
	ProductID    uuid.UUID  `json:"product_id"`
	CustomerID   uuid.UUID  `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	AvatarURL    *string    `json:"avatar_url"`
	OrderID      uuid.UUID  `json:"order_id"`
	Rating       int        `json:"rating"`
	Title        *string    `json:"title"`
	Comment      *string    `json:"comment"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type listReviewsResponse struct {
	Reviews       []reviewListItemResponse `json:"reviews"`
	AverageRating float64                  `json:"average_rating"`
	ReviewCount   int                      `json:"review_count"`
	Page          int                      `json:"page"`
	Limit         int                      `json:"limit"`
	Total         int                      `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}
