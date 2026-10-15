package reviewhttp

import (
	"time"

	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type createReviewRequest struct {
	Rating  int        `json:"rating"`
	Title   *string    `json:"title"`
	Comment *string    `json:"comment"`
	OrderID *uuid.UUID `json:"order_id"`
}

type reviewResponse struct {
	ID           uuid.UUID  `json:"id"`
	ProductID    uuid.UUID  `json:"product_id"`
	CustomerID   uuid.UUID  `json:"customer_id"`
	OrderID      uuid.UUID  `json:"order_id"`
	Rating       int        `json:"rating"`
	Title        *string    `json:"title"`
	Comment      *string    `json:"comment"`
	HelpfulCount int        `json:"helpful_count"`
	Media        []string   `json:"media"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
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
	HelpfulCount int        `json:"helpful_count"`
	Media        []string   `json:"media"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type RatingSummaryResponse struct {
	AverageRating float64 `json:"average_rating"`
	ReviewsCount  int     `json:"reviews_count"`
}

type listReviewsResponse struct {
	Items         []reviewListItemResponse  `json:"items"`
	Pagination    pagination.PaginationMeta `json:"pagination"`
	RatingSummary RatingSummaryResponse     `json:"rating_summary"`
}

type messageResponse struct {
	Message string `json:"message"`
}
