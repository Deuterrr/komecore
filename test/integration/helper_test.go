//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func skipIfDockerUnavailable(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "docker") ||
			strings.Contains(errStr, "pipe") ||
			strings.Contains(errStr, "daemon") ||
			strings.Contains(errStr, "provider") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "cannot connect") ||
			strings.Contains(errStr, "the system cannot find the file specified") {
			t.Skipf("Skipping integration test: Docker daemon is unavailable: %v", err)
		}
	}
}

func seedUserAndCustomer(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	userID := uuid.New()
	accountID := uuid.New()
	customerID := uuid.New()
	now := time.Now().UTC()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, phone, created_at, updated_at)
		VALUES ($1, 'Test Buyer', '+628123456789', $2, $2);
	`, userID, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("insert user: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO accounts (id, user_id, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, '$2a$10$abcdefghijklmnopqrstuvwxyz123456', 'customer', true, $4, $4);
	`, accountID, userID, fmt.Sprintf("buyer-%s@example.com", customerID.String()[:8]), now)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("insert account: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO customers (id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $3);
	`, customerID, userID, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("insert customer: %w", err)
	}

	return userID, accountID, customerID, nil
}

func seedShopAndProduct(ctx context.Context, pool *pgxpool.Pool, initialStock int) (uuid.UUID, uuid.UUID, error) {
	shopID := uuid.New()
	productID := uuid.New()
	inventoryID := uuid.New()
	now := time.Now().UTC()

	shopSlug := fmt.Sprintf("shop-%s", shopID.String()[:8])
	_, err := pool.Exec(ctx, `
		INSERT INTO shops (id, name, slug, description, is_active, created_at, updated_at)
		VALUES ($1, 'Florist Boutique', $2, 'Premium floral arrangements', true, $3, $3);
	`, shopID, shopSlug, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert shop: %w", err)
	}

	prodSlug := fmt.Sprintf("rose-bouquet-%s", productID.String()[:8])
	_, err = pool.Exec(ctx, `
		INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at, updated_at)
		VALUES ($1, $2, 'Red Rose Bouquet', $3, 'Freshly cut long-stem red roses', 'active', 150000, 500, $4, $4);
	`, productID, fmt.Sprintf("SKU-%s", productID.String()[:8]), prodSlug, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert product: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, $5, $5);
	`, inventoryID, productID, shopID, initialStock, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert inventory: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO product_performance (product_id, sales_count, views_count, average_rating, reviews_count, updated_at)
		VALUES ($1, 0, 0, 0.0, 0, $2)
		ON CONFLICT (product_id) DO NOTHING;
	`, productID, now)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert product performance: %w", err)
	}

	return shopID, productID, nil
}
