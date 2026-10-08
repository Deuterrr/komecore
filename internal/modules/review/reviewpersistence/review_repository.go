package reviewpersistence

import (
	"context"
	"errors"
	"fmt"
	"math"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/review/reviewdomain"
	"komecore/internal/modules/review/reviewrepo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReviewRepository struct{}

func NewReviewRepository() *ReviewRepository {
	return &ReviewRepository{}
}

func (r *ReviewRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	review *reviewdomain.Review,
) error {
	query := `
		INSERT INTO product_reviews (
			id, product_id, customer_id, order_id, rating, title, comment, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	_, err := exec.Exec(
		ctx,
		query,
		review.ID,
		review.ProductID,
		review.CustomerID,
		review.OrderID,
		review.Rating,
		review.Title,
		review.Comment,
		review.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create review failed: %w", err)
	}

	return nil
}

func (r *ReviewRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*reviewdomain.Review, error) {
	query := `
		SELECT
			id, product_id, customer_id, order_id, rating, title, comment, created_at, updated_at
		FROM product_reviews
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var item reviewdomain.Review
	err := exec.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.ProductID,
		&item.CustomerID,
		&item.OrderID,
		&item.Rating,
		&item.Title,
		&item.Comment,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query review by id failed: %w", err)
	}

	return &item, nil
}

func (r *ReviewRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE product_reviews
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete review failed: %w", err)
	}

	return nil
}

func (r *ReviewRepository) ListByProductID(
	ctx context.Context,
	exec transaction.Executor,
	params reviewrepo.ListReviewsParams,
) ([]reviewdomain.ReviewWithCustomer, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM product_reviews
		WHERE product_id = $1 AND deleted_at IS NULL
	`

	var total int
	if err := exec.QueryRow(ctx, countQuery, params.ProductID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count product reviews failed: %w", err)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `
		SELECT
			r.id,
			r.product_id,
			r.customer_id,
			r.order_id,
			r.rating,
			r.title,
			r.comment,
			r.created_at,
			r.updated_at,
			u.name,
			u.avatar_url
		FROM product_reviews r
		JOIN customers c ON r.customer_id = c.id
		JOIN users u ON c.user_id = u.id
		WHERE r.product_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := exec.Query(ctx, query, params.ProductID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query product reviews failed: %w", err)
	}
	defer rows.Close()

	var results []reviewdomain.ReviewWithCustomer
	for rows.Next() {
		var item reviewdomain.ReviewWithCustomer
		if err := rows.Scan(
			&item.Review.ID,
			&item.Review.ProductID,
			&item.Review.CustomerID,
			&item.Review.OrderID,
			&item.Review.Rating,
			&item.Review.Title,
			&item.Review.Comment,
			&item.Review.CreatedAt,
			&item.Review.UpdatedAt,
			&item.CustomerName,
			&item.AvatarURL,
		); err != nil {
			return nil, 0, fmt.Errorf("scan product review failed: %w", err)
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate product reviews failed: %w", err)
	}

	return results, total, nil
}

func (r *ReviewRepository) GetRatingSummary(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
) (*reviewdomain.ProductRatingSummary, error) {
	query := `
		SELECT
			COALESCE(AVG(rating::float8), 0.0),
			COUNT(*)
		FROM product_reviews
		WHERE product_id = $1 AND deleted_at IS NULL
	`

	var rawAvg float64
	var count int
	if err := exec.QueryRow(ctx, query, productID).Scan(&rawAvg, &count); err != nil {
		return nil, fmt.Errorf("query rating summary failed: %w", err)
	}

	summary := &reviewdomain.ProductRatingSummary{
		AverageRating: math.Round(rawAvg*100) / 100,
		ReviewCount:   count,
	}

	return summary, nil
}

func (r *ReviewRepository) HasReviewedOrder(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
	productID uuid.UUID,
	orderID uuid.UUID,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM product_reviews
			WHERE customer_id = $1 AND product_id = $2 AND order_id = $3 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := exec.QueryRow(ctx, query, customerID, productID, orderID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check order review status failed: %w", err)
	}

	return exists, nil
}

func (r *ReviewRepository) GetReviewedOrderIDs(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
	productID uuid.UUID,
) ([]uuid.UUID, error) {
	query := `
		SELECT order_id
		FROM product_reviews
		WHERE customer_id = $1 AND product_id = $2 AND deleted_at IS NULL
	`

	rows, err := exec.Query(ctx, query, customerID, productID)
	if err != nil {
		return nil, fmt.Errorf("query reviewed order ids failed: %w", err)
	}
	defer rows.Close()

	var orderIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan reviewed order id failed: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reviewed order ids failed: %w", err)
	}

	return orderIDs, nil
}
