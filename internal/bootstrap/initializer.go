package bootstrap

import (
	"context"
	"fmt"
	"log"

	"komecore/internal/config"
	database "komecore/internal/infra/db"
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
		if err := database.RunMigration(i.dbCfg); err != nil {
			return fmt.Errorf("failed to apply migrations during startup: %w", err)
		}
	}

	// Synchronize payment methods
	if err := i.c.SyncPaymentMethods.Execute(ctx); err != nil {
		return fmt.Errorf("failed to sync payment methods on startup: %w", err)
	}

	return nil
}
