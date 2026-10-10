//go:build integration

package integration_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"komecore/internal/testutil/testdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedTables = []string{
	"accounts",
	"carts",
	"cart_items",
	"coupons",
	"coupon_redemptions",
	"couriers",
	"customer_addresses",
	"customers",
	"inventory",
	"invoices",
	"invoice_items",
	"oauth_connections",
	"orders",
	"order_items",
	"outbox_events",
	"payments",
	"payment_channel_data",
	"payment_events",
	"payment_instructions",
	"payment_methods",
	"payment_webhook_events",
	"products",
	"product_images",
	"product_performance",
	"product_reviews",
	"product_stock_history",
	"refresh_tokens",
	"roles",
	"seed_versions",
	"sessions",
	"shipments",
	"shops",
	"shop_addresses",
	"staff",
	"staff_memberships",
	"users",
	"verification_challenges",
	"wishlists",
}

func getPublicTables(ctx context.Context, t *testing.T, tdb *testdb.TestDB) []string {
	t.Helper()
	rows, err := tdb.Pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		  AND table_name NOT IN ('schema_migrations')
		ORDER BY table_name;
	`)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		tables = append(tables, name)
	}
	require.NoError(t, rows.Err())
	return tables
}

func TestPostgresReversibleMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Provision Ephemeral Postgres 17 Alpine
	t.Log("Provisioning ephemeral PostgreSQL 17 container...")
	tdb, err := testdb.NewTestDB(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err, "must be able to provision ephemeral postgres testcontainer")
	defer func() {
		_ = tdb.Close(ctx)
	}()

	// Verify Fresh Database has zero tables initially
	initialTables := getPublicTables(ctx, t, tdb)
	assert.Empty(t, initialTables, "fresh database must contain 0 tables")

	// Step UP: Apply all consolidated migrations (0001 -> 0012)
	t.Log("Applying migrations UP (0001 -> 0012)...")
	err = tdb.ApplyMigrations()
	require.NoError(t, err, "initial migration UP must succeed cleanly")

	// Verify all expected tables exist
	appliedTables := getPublicTables(ctx, t, tdb)
	sort.Strings(appliedTables)
	expectedSorted := make([]string, len(expectedTables))
	copy(expectedSorted, expectedTables)
	sort.Strings(expectedSorted)

	for _, expected := range expectedSorted {
		assert.Contains(t, appliedTables, expected, "table %s must exist after migration UP", expected)
	}
	assert.GreaterOrEqual(t, len(appliedTables), len(expectedSorted), "all domain tables must exist")

	// Verify migration version is 12 and not dirty
	var version int64
	var dirty bool
	err = tdb.Pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations LIMIT 1;").Scan(&version, &dirty)
	require.NoError(t, err)
	assert.Equal(t, int64(12), version, "schema_migrations version must be 12")
	assert.False(t, dirty, "migration must not be marked dirty")

	// Step DOWN: Roll back all migrations to version 0 (0012 -> 0001)
	t.Log("Rolling back all migrations DOWN (0012 -> 0001)...")
	err = tdb.RollbackAll()
	require.NoError(t, err, "migration DOWN must succeed cleanly without FK or cascade errors")

	// Verify all application tables are dropped
	rolledBackTables := getPublicTables(ctx, t, tdb)
	assert.Empty(t, rolledBackTables, "all tables must be dropped after full rollback")

	// Step RE-UP: Re-apply all migrations (0001 -> 0012)
	t.Log("Re-applying migrations UP (0001 -> 0012)...")
	err = tdb.ApplyMigrations()
	require.NoError(t, err, "re-applying migration UP after rollback must succeed")

	// Verify all expected tables exist once again
	reAppliedTables := getPublicTables(ctx, t, tdb)
	for _, expected := range expectedSorted {
		assert.Contains(t, reAppliedTables, expected, "table %s must exist after re-applying UP", expected)
	}

	t.Log("Reversible migration test passed successfully across all 12 migration files!")
}
