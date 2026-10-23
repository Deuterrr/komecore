package cartpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CartRepository struct{}

func NewCartRepository() *CartRepository {
	return &CartRepository{}
}

func (r *CartRepository) GetWithItemsByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cart.Cart, error) {
	query := `
		SELECT
			c.id,
			c.customer_id,
			c.created_at,
			c.updated_at,
			ci.id,
			ci.product_id,
			ci.shop_id,
			ci.quantity,
			ci.item_options
		FROM carts c
		LEFT JOIN
			cart_items ci ON ci.cart_id = c.id
			AND ci.deleted_at IS NULL
		WHERE c.customer_id = $1
		ORDER BY ci.created_at
	`

	rows, err := exec.Query(ctx, query, customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query cart with items by customer id failed: %w", err)
	}
	defer rows.Close()

	var c *cart.Cart
	for rows.Next() {
		var (
			cID        uuid.UUID
			custID     uuid.UUID
			createdAt  time.Time
			updatedAt  *time.Time
			itemID     *uuid.UUID
			productID  *uuid.UUID
			shopID     *uuid.UUID
			quantity   *int
			rawOptions []byte
		)

		err := rows.Scan(
			&cID,
			&custID,
			&createdAt,
			&updatedAt,
			&itemID,
			&productID,
			&shopID,
			&quantity,
			&rawOptions,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping cart with items model to domain failed: %w", err)
		}

		if c == nil {
			c = &cart.Cart{
				ID:         cID,
				CustomerID: custID,
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
				Items:      []cart.CartItem{},
			}
		}

		if itemID != nil {
			var itemOptions cart.ItemOptions
			if len(rawOptions) > 0 {
				_ = json.Unmarshal(rawOptions, &itemOptions)
			}

			pID := uuid.Nil
			if productID != nil {
				pID = *productID
			}

			c.Items = append(c.Items, cart.CartItem{
				ID:          *itemID,
				ProductID:   pID,
				ShopID:      *shopID,
				Quantity:    *quantity,
				ItemOptions: itemOptions.Normalized(),
			})
		}
	}

	if c == nil {
		return nil, nil
	}

	return c, nil
}

func (r *CartRepository) NewCart(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*cart.Cart, error) {
	query := `
		INSERT INTO carts (
			customer_id
		)
		VALUES ($1)
		ON CONFLICT (customer_id)
		DO UPDATE SET updated_at = NOW()
		RETURNING
			id,
			customer_id,
			created_at,
			updated_at
	`

	var c cart.Cart
	err := exec.QueryRow(ctx, query, customerID).Scan(
		&c.ID,
		&c.CustomerID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(transaction.MapDBError(err), transaction.ErrUniqueViolation) {
			existing, getErr := r.GetWithItemsByCustomerID(ctx, exec, customerID)
			if getErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("insert cart failed: %w", err)
	}

	c.Items = []cart.CartItem{}

	return &c, nil
}

func (r *CartRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	cart *cart.Cart,
) error {
	const insertItemQuery = `
		INSERT INTO cart_items (
			id,
			cart_id,
			product_id,
			shop_id,
			quantity,
			item_options
		)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)
		ON CONFLICT (id)
		DO UPDATE SET
			shop_id      = EXCLUDED.shop_id,
			quantity     = EXCLUDED.quantity,
			item_options = EXCLUDED.item_options,
			updated_at   = NOW()
	`

	const softDeleteByIDQuery = `
		UPDATE cart_items
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
	`

	for i := range cart.Items {
		item := &cart.Items[i]
		if item.DeletedAt != nil {
			if _, err := exec.Exec(ctx, softDeleteByIDQuery, item.ID); err != nil {
				return fmt.Errorf("soft-delete cart item failed: %w", err)
			}
			continue
		}

		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}

		optBytes, _ := json.Marshal(item.ItemOptions.Normalized())
		args := []any{
			item.ID,
			cart.ID,
			item.ProductID,
			item.ShopID,
			item.Quantity,
			string(optBytes),
		}

		if _, err := exec.Exec(ctx, insertItemQuery, args...); err != nil {
			return fmt.Errorf("insert cart item failed: %w", err)
		}
	}

	const updateCartQuery = `
		UPDATE carts
		SET
			updated_at = NOW()
		WHERE id = $1
	`

	if _, err := exec.Exec(ctx, updateCartQuery, cart.ID); err != nil {
		return fmt.Errorf("update cart failed: %w", err)
	}

	return nil
}

func (r *CartRepository) DeleteByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) error {
	query := `
		UPDATE cart_items ci
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		FROM carts c
		WHERE ci.cart_id = c.id
			AND c.customer_id = $1
			AND ci.deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("delete customer cart items failed: %w", err)
	}

	return nil
}
