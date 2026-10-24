package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	database "komecore/internal/infra/db"
)

// runSeed discovers and applies SQL seed scripts from root seeds/<dialect>/ and optionally seeds/<dialect>/dev/.
func runSeed(conn *database.Connection, withDev bool, force bool, appEnv string) error {
	if conn == nil {
		return fmt.Errorf("database: connection is nil")
	}

	if appEnv == "" {
		appEnv = os.Getenv("APP_ENV")
	}
	if appEnv == "" {
		appEnv = "development"
	}

	// Safety guard against applying dev seeds to production
	if withDev && strings.EqualFold(appEnv, "production") && !force {
		return fmt.Errorf("safety violation: cannot apply development seeds to production environment (use --force to override)")
	}

	dialect := strings.ToLower(conn.Dialect)
	if dialect == "" {
		dialect = "postgres"
	}

	seedsDir, err := findSeedsDir(dialect)
	if err != nil {
		return fmt.Errorf("resolve seeds directory: %w", err)
	}

	ctx := context.Background()

	if err := runSeedDirectory(ctx, conn, seedsDir, ""); err != nil {
		return fmt.Errorf("base seeds failed: %w", err)
	}

	if withDev {
		devDir := filepath.Join(seedsDir, "dev")
		if fi, err := os.Stat(devDir); err == nil && fi.IsDir() {
			log.Printf("database: applying development seeds")
			if err := runSeedDirectory(ctx, conn, devDir, "dev/"); err != nil {
				return fmt.Errorf("development seeds failed: %w", err)
			}
		}
	}

	log.Printf("database: seeding completed successfully")
	return nil
}

// findSeedsDir resolves the directory containing SQL seed files for the given dialect.
func findSeedsDir(dialect string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, "seeds", dialect)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find seeds/%s directory in any parent directory", dialect)
		}
		dir = parent
	}
}

func runSeedDirectory(ctx context.Context, conn *database.Connection, dir string, prefix string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory %s: %w", dir, err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		versionName := prefix + file
		seeded, err := isSeedVersionApplied(ctx, conn, versionName)
		if err != nil {
			return fmt.Errorf("check seed version %s: %w", versionName, err)
		}

		if seeded {
			log.Printf("database: %s already seeded, skipping", versionName)
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return fmt.Errorf("read seed file %s: %w", file, err)
		}

		log.Printf("database: seeding %s", versionName)
		if err := executeSeedSQL(ctx, conn, string(content)); err != nil {
			return fmt.Errorf("execute seed file %s: %w", file, err)
		}

		if err := markSeedVersionApplied(ctx, conn, versionName); err != nil {
			return fmt.Errorf("record seed version %s: %w", versionName, err)
		}
	}

	return nil
}

func isSeedVersionApplied(ctx context.Context, conn *database.Connection, name string) (bool, error) {
	if conn.Pool != nil {
		var exists bool
		err := conn.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM seed_versions WHERE name = $1
			)
		`, name).Scan(&exists)
		return exists, err
	}

	if conn.SqlDB != nil {
		var count int
		err := conn.SqlDB.QueryRowContext(ctx, `
			SELECT COUNT(1) FROM seed_versions WHERE name = @p1
		`, name).Scan(&count)
		return count > 0, err
	}

	return false, fmt.Errorf("no active connection pool available")
}

func executeSeedSQL(ctx context.Context, conn *database.Connection, sqlContent string) error {
	if strings.TrimSpace(sqlContent) == "" {
		return nil
	}

	if conn.Pool != nil {
		tx, err := conn.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		if _, err := tx.Exec(ctx, sqlContent); err != nil {
			return fmt.Errorf("exec sql: %w", err)
		}

		return tx.Commit(ctx)
	}

	if conn.SqlDB != nil {
		tx, err := conn.SqlDB.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin sqlserver tx: %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, sqlContent); err != nil {
			return fmt.Errorf("exec sqlserver sql: %w", err)
		}

		return tx.Commit()
	}

	return fmt.Errorf("no active connection available for sql execution")
}

func markSeedVersionApplied(ctx context.Context, conn *database.Connection, name string) error {
	if conn.Pool != nil {
		_, err := conn.Pool.Exec(ctx, `
			INSERT INTO seed_versions (name, version)
			VALUES ($1, '1.0')
			ON CONFLICT (name) DO NOTHING
		`, name)
		return err
	}

	if conn.SqlDB != nil {
		_, err := conn.SqlDB.ExecContext(ctx, `
			IF NOT EXISTS (SELECT 1 FROM seed_versions WHERE name = @p1)
				INSERT INTO seed_versions (name, version) VALUES (@p1, '1.0')
		`, name)
		return err
	}

	return nil
}
