package bootstrap

import (
	"context"
	"testing"

	"komecore/internal/config"
)

func TestInitializer_NilContainer(t *testing.T) {
	init := NewInitializer(nil, config.AppConfig{}, config.DatabaseConfig{})
	if err := init.Run(context.Background()); err != nil {
		t.Errorf("expected no error for nil container, got %v", err)
	}
}

func TestInitializer_AutoMigrate_MissingDSN(t *testing.T) {
	appCfg := config.AppConfig{AutoMigrate: true}
	dbCfg := config.DatabaseConfig{} // DSN is nil

	init := NewInitializer(&Container{}, appCfg, dbCfg)
	err := init.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when AutoMigrate is true and DSN is not configured, got nil")
	}
}
