package persistence

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/repository"

	"github.com/google/uuid"
)

type wishlistRepositoryImpl struct{}

func NewWishlistRepositoryImpl() repository.WishlistRepository {
	return &wishlistRepositoryImpl{}
}

func (r *wishlistRepositoryImpl) Add(
	ctx context.Context,
	exec transaction.Executor,
	item domain.WishlistItem,
) error {
	query := `
		INSERT INTO wishlists (customer_id, product_id, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := exec.Exec(ctx, query, item.CustomerID, item.ProductID, item.CreatedAt)
	if err != nil {
		return fmt.Errorf("add to wishlist failed: %w", err)
	}

	return nil
}

func (r *wishlistRepositoryImpl) Remove(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
	productID uuid.UUID,
) error {
	query := `
		DELETE FROM wishlists
		WHERE customer_id = $1 AND product_id = $2
	`

	_, err := exec.Exec(ctx, query, customerID, productID)
	if err != nil {
		return fmt.Errorf("remove from wishlist failed: %w", err)
	}

	return nil
}

func (r *wishlistRepositoryImpl) ListByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) ([]domain.WishlistItem, error) {
	query := `
		SELECT customer_id, product_id, created_at
		FROM wishlists
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := exec.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("list wishlist items failed: %w", err)
	}
	defer rows.Close()

	var items []domain.WishlistItem
	for rows.Next() {
		var item domain.WishlistItem
		if err := rows.Scan(&item.CustomerID, &item.ProductID, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan wishlist item failed: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wishlist items failed: %w", err)
	}

	return items, nil
}

func (r *wishlistRepositoryImpl) Exists(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
	productID uuid.UUID,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM wishlists
			WHERE customer_id = $1 AND product_id = $2
		)
	`

	var exists bool
	err := exec.QueryRow(ctx, query, customerID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check wishlist item existence failed: %w", err)
	}

	return exists, nil
}
