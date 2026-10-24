package main

import (
	"os"
	"path/filepath"
	"testing"

	"komecore/internal/config"

	"github.com/golang-migrate/migrate/v4/source/file"
)

func findSQLDialectDir(t *testing.T, dialect string) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	for {
		migPath := filepath.Join(dir, "migrations", dialect)
		if fi, err := os.Stat(migPath); err == nil && fi.IsDir() {
			return migPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find migrations/%s directory in any parent directory", dialect)
		}
		dir = parent
	}
}

func TestPostgresMigrationsIntegrity(t *testing.T) {
	testMigrationsIntegrity(t, "postgres")
}

func TestSQLServerMigrationsIntegrity(t *testing.T) {
	testMigrationsIntegrity(t, "sqlserver")
}

func testMigrationsIntegrity(t *testing.T, dialect string) {
	t.Helper()

	migDir := findSQLDialectDir(t, dialect)

	expectedMigrations := []string{
		"0001_create_system_and_auth_tables",
		"0002_create_shops_and_staff_tables",
		"0003_create_products_and_inventory_tables",
		"0004_create_addresses_and_couriers_tables",
		"0005_create_carts_tables",
		"0006_create_orders_and_invoices_tables",
		"0007_create_shipments_tables",
		"0008_create_payments_tables",
		"0009_create_reviews_tables",
		"0010_create_wishlists_tables",
		"0011_create_coupons_and_search_vector",
		"0012_create_outbox_events_table",
	}

	for _, name := range expectedMigrations {
		upFile := filepath.Join(migDir, name+".up.sql")
		if _, err := os.Stat(upFile); os.IsNotExist(err) {
			t.Errorf("[%s] missing migration up file: %s", dialect, upFile)
		}

		downFile := filepath.Join(migDir, name+".down.sql")
		if _, err := os.Stat(downFile); os.IsNotExist(err) {
			t.Errorf("[%s] missing migration down file: %s", dialect, downFile)
		}
	}

	migURL := "file://" + filepath.ToSlash(migDir)
	driver, err := (&file.File{}).Open(migURL)
	if err != nil {
		t.Fatalf("golang-migrate file driver failed to open %s: %v", migURL, err)
	}
	defer driver.Close()

	firstVersion, err := driver.First()
	if err != nil {
		t.Fatalf("failed to get first migration version: %v", err)
	}
	if firstVersion != 1 {
		t.Errorf("expected first migration version to be 1, got %d", firstVersion)
	}

	currentVersion := firstVersion
	count := 1
	for {
		nextVersion, err := driver.Next(currentVersion)
		if err != nil {
			break
		}
		count++
		currentVersion = nextVersion
	}

	if count != 12 {
		t.Errorf("[%s] expected 12 sequential migration versions, got %d", dialect, count)
	}
}

func TestGetMigrationVersion_UnconfiguredDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Dialect: "postgres",
		DSN:     nil,
	}
	_, _, err := getMigrationVersion(cfg)
	if err == nil {
		t.Fatal("expected error when DSN is nil, got nil")
	}
}
