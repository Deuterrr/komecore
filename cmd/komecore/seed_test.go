package main

import (
	"os"
	"path/filepath"
	"testing"

	database "komecore/internal/infra/db"
)

func TestSeedRunner_FindSeedsDir(t *testing.T) {
	postgresDir, err := findSeedsDir("postgres")
	if err != nil {
		t.Fatalf("failed to find postgres seeds dir: %v", err)
	}

	expectedPostgresFiles := []string{
		"0001_roles.sql",
		"0002_admin.sql",
		"0003_couriers.sql",
		"0004_payment_instructions.sql",
	}

	for _, file := range expectedPostgresFiles {
		path := filepath.Join(postgresDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected seed file %s does not exist", path)
		}
	}

	devDir := filepath.Join(postgresDir, "dev")
	expectedDevFiles := []string{
		"0001_shops.sql",
		"0002_products.sql",
		"0003_customers.sql",
	}
	for _, file := range expectedDevFiles {
		path := filepath.Join(devDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected dev seed file %s does not exist", path)
		}
	}

	sqlserverDir, err := findSeedsDir("sqlserver")
	if err != nil {
		t.Fatalf("failed to find sqlserver seeds dir: %v", err)
	}
	for _, file := range expectedPostgresFiles {
		path := filepath.Join(sqlserverDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected sqlserver seed file %s does not exist", path)
		}
	}
}

func TestSeedRunner_NilConnection(t *testing.T) {
	err := runSeed(nil, false, false, "development")
	if err == nil {
		t.Fatal("expected error when connection is nil, got nil")
	}
}

func TestSeedRunner_ProductionSafetyViolation(t *testing.T) {
	// withDev in production without force should fail immediately before DB call
	conn := &database.Connection{Dialect: "postgres"}
	err := runSeed(conn, true, false, "production")
	if err == nil {
		t.Fatal("expected safety violation error when running dev seeds in production, got nil")
	}
}
