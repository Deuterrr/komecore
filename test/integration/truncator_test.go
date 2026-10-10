//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"komecore/test/testutil/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncator_CleanStateIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tdb, err := testdb.NewTestDB(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err)
	defer func() {
		_ = tdb.Close(ctx)
	}()

	err = tdb.ApplyMigrations()
	require.NoError(t, err)

	// 1. Insert seed data into parent-child hierarchy (shops -> products -> inventory)
	shopID := uuid.New().String()
	productID := uuid.New().String()
	now := time.Now()

	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO shops (id, name, slug, description, is_active, created_at, updated_at)
		VALUES ($1, 'Test Florist', 'test-florist', 'A cozy florist shop', true, $2, $2);
	`, shopID, now)
	require.NoError(t, err)

	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO products (id, sku, name, slug, description, status, base_price, weight, created_at, updated_at)
		VALUES ($1, 'SKU-ROSE-001', 'Red Rose Bouquet', 'red-rose-bouquet', 'Fresh red roses', 'active', 150000, 500, $2, $2);
	`, productID, now)
	require.NoError(t, err)

	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO inventory (id, product_id, shop_id, stock, reserved_stock, created_at, updated_at)
		VALUES ($1, $2, $3, 25, 0, $4, $4);
	`, uuid.New().String(), productID, shopID, now)
	require.NoError(t, err)

	// Verify rows exist
	var shopCount, productCount, inventoryCount int
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM shops;").Scan(&shopCount)
	require.NoError(t, err)
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM products;").Scan(&productCount)
	require.NoError(t, err)
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM inventory;").Scan(&inventoryCount)
	require.NoError(t, err)

	assert.Equal(t, 1, shopCount)
	assert.Equal(t, 1, productCount)
	assert.Equal(t, 1, inventoryCount)

	// 2. Execute Truncate
	err = tdb.Truncate(ctx)
	require.NoError(t, err, "truncating tables should succeed without error")

	// 3. Verify all tables are empty
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM shops;").Scan(&shopCount)
	require.NoError(t, err)
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM products;").Scan(&productCount)
	require.NoError(t, err)
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM inventory;").Scan(&inventoryCount)
	require.NoError(t, err)

	assert.Equal(t, 0, shopCount, "shops table must be empty after truncate")
	assert.Equal(t, 0, productCount, "products table must be empty after truncate")
	assert.Equal(t, 0, inventoryCount, "inventory table must be empty after truncate")

	// 4. Verify re-insert succeeds cleanly (validating sequence reset and clean state)
	newShopID := uuid.New().String()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO shops (id, name, slug, description, is_active, created_at, updated_at)
		VALUES ($1, 'Second Shop', 'second-shop', 'Second florist shop', true, $2, $2);
	`, newShopID, now)
	require.NoError(t, err, "insert after truncate must succeed")

	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM shops;").Scan(&shopCount)
	require.NoError(t, err)
	assert.Equal(t, 1, shopCount)
}
