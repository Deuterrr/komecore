package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/file"
)

func findSQLDialectDir(t *testing.T, dialect string) string {
	t.Helper()

	// Walk upwards until we find migrations/<dialect>
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
	}

	// 1. Verify all expected migration pairs exist and are non-empty
	for _, base := range expectedMigrations {
		for _, direction := range []string{"up", "down"} {
			filename := base + "." + direction + ".sql"
			fullPath := filepath.Join(migDir, filename)

			info, err := os.Stat(fullPath)
			if err != nil {
				t.Errorf("missing migration file: %s", filename)
				continue
			}

			if info.Size() == 0 {
				t.Errorf("migration file is empty: %s", filename)
			}

			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Errorf("failed to read migration file %s: %v", filename, err)
				continue
			}

			sql := string(content)
			if direction == "up" {
				// Must contain CREATE statements
				if !strings.Contains(sql, "CREATE") {
					t.Errorf("migration up file %s should contain CREATE statements", filename)
				}
				// Postgres-specific: all timestamp columns must use TIMESTAMPTZ
				if dialect == "postgres" {
					lines := strings.Split(sql, "\n")
					for lineNum, line := range lines {
						upper := strings.ToUpper(line)
						for _, field := range strings.Fields(upper) {
							trimmed := strings.Trim(field, "();,")
							if trimmed == "TIMESTAMP" {
								t.Errorf("%s:%d uses TIMESTAMP instead of standard TIMESTAMPTZ: %s", filename, lineNum+1, strings.TrimSpace(line))
							}
						}
					}
				}
				// SQL Server-specific: must use DATETIMEOFFSET, not TIMESTAMPTZ
				if dialect == "sqlserver" {
					if strings.Contains(strings.ToUpper(sql), "TIMESTAMPTZ") {
						t.Errorf("migration up file %s uses TIMESTAMPTZ which is not valid SQL Server syntax", filename)
					}
				}
			} else {
				// Must contain DROP statements
				if !strings.Contains(sql, "DROP") {
					t.Errorf("migration down file %s should contain DROP statements", filename)
				}
			}
		}
	}

	// 2. Verify exactly 16 migration files exist in total (8 up, 8 down)
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	sqlCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlCount++
		}
	}

	if sqlCount != 20 {
		t.Errorf("[%s] expected exactly 20 .sql migration files (10 up, 10 down), found %d", dialect, sqlCount)
	}

	// 3. Verify golang-migrate file driver can parse the migrations directory
	migURL := "file://" + filepath.ToSlash(migDir)
	fDriver := &file.File{}
	driver, err := fDriver.Open(migURL)
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

	if count != 10 {
		t.Errorf("[%s] expected 10 sequential migration versions, got %d", dialect, count)
	}
}
