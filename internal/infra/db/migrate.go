package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"komecore/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlserver"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func resolveMigrationSourceURL(cfg config.DatabaseConfig) string {
	src := cfg.MigrationDir()
	if strings.HasPrefix(src, "file://") || strings.HasPrefix(src, "file:") {
		return src
	}

	if filepath.IsAbs(src) {
		wd, err := os.Getwd()
		if err == nil {
			if rel, err := filepath.Rel(wd, src); err == nil {
				return "file://" + filepath.ToSlash(rel)
			}
		}
		// Fallback to opaque URL
		return "file:" + filepath.ToSlash(src)
	}

	return "file://" + filepath.ToSlash(src)
}

func RunMigration(cfg config.DatabaseConfig) error {
	log.Printf("database: running migration [dialect=%s]", cfg.Dialect)

	dsn := *cfg.DSN
	m, err := migrate.New(resolveMigrationSourceURL(cfg), dsn)
	if err != nil {
		return fmt.Errorf("failed to init migration: %w", err)
	}

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("database: already up to date")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Printf("database: migration applied")
	return nil
}

func RunRollback(cfg config.DatabaseConfig) error {
	log.Printf("database: running rollback (1 step) [dialect=%s]", cfg.Dialect)

	dsn := *cfg.DSN
	m, err := migrate.New(resolveMigrationSourceURL(cfg), dsn)
	if err != nil {
		return fmt.Errorf("failed to init migration: %w", err)
	}

	err = m.Steps(-1)
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("database: already at earliest version")
			return nil
		}
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Printf("database: rollback applied")
	return nil
}

func RunMigrationDown(cfg config.DatabaseConfig) error {
	log.Printf("database: running rollback (all steps) [dialect=%s]", cfg.Dialect)

	dsn := *cfg.DSN
	m, err := migrate.New(resolveMigrationSourceURL(cfg), dsn)
	if err != nil {
		return fmt.Errorf("failed to init migration: %w", err)
	}

	err = m.Down()
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("database: already at earliest version")
			return nil
		}
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Printf("database: all migrations rolled back")
	return nil
}
