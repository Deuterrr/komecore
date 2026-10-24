package bootstrap

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"komecore/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlserver"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Initializer handles pre-flight application startup tasks
// before traffic is served.
type Initializer struct {
	c      *Container
	dbCfg  config.DatabaseConfig
	appCfg config.AppConfig
}

func NewInitializer(c *Container, appCfg config.AppConfig, dbCfg config.DatabaseConfig) *Initializer {
	return &Initializer{
		c:      c,
		appCfg: appCfg,
		dbCfg:  dbCfg,
	}
}

func (i *Initializer) Run(ctx context.Context) error {
	if i == nil || i.c == nil {
		return nil
	}

	// Run migrations if enabled or in development
	if i.appCfg.AutoMigrate {
		if i.dbCfg.DSN == nil || *i.dbCfg.DSN == "" {
			return fmt.Errorf("failed to apply migrations during startup: database DSN is not configured")
		}
		log.Printf("running pre-flight database migrations [dialect=%s]", i.dbCfg.Dialect)
		if err := runPreflightMigration(i.dbCfg); err != nil {
			return fmt.Errorf("failed to apply migrations during startup: %w", err)
		}
	}

	// Synchronize payment methods
	if i.c.Payment != nil {
		if err := i.c.Payment.SyncPaymentMethods(ctx); err != nil {
			return fmt.Errorf("failed to sync payment methods on startup: %w", err)
		}
	}

	return nil
}

func runPreflightMigration(cfg config.DatabaseConfig) error {
	if cfg.DSN == nil || *cfg.DSN == "" {
		return fmt.Errorf("database DSN is not configured")
	}

	src := cfg.MigrationDir()
	if !strings.HasPrefix(src, "file://") && !strings.HasPrefix(src, "file:") {
		src = "file://" + filepath.ToSlash(src)
	}

	m, err := migrate.New(src, *cfg.DSN)
	if err != nil {
		return fmt.Errorf("failed to init migration: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
