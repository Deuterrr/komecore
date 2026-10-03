package config

import (
	"testing"
)

func TestResolveDBConfig(t *testing.T) {
	t.Run("resolves postgres target", func(t *testing.T) {
		t.Setenv("POSTGRES_DB_HOST", "127.0.0.1")
		t.Setenv("POSTGRES_DB_PORT", "5432")
		t.Setenv("POSTGRES_DB_USER", "komee")
		t.Setenv("POSTGRES_DB_PASSWORD", "password")
		t.Setenv("POSTGRES_DB_NAME", "komecore")
		t.Setenv("POSTGRES_DB_SSLMODE", "disable")

		cfg := ResolveDBConfig("postgres")
		if cfg.Host != "127.0.0.1" {
			t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
		}
		expectedDSN := "postgresql://komee:password@127.0.0.1:5432/komecore?sslmode=disable"
		if cfg.DSN == nil || *cfg.DSN != expectedDSN {
			t.Errorf("expected DSN %s, got %v", expectedDSN, cfg.DSN)
		}
		if cfg.Dialect != "postgres" {
			t.Errorf("expected dialect postgres, got %s", cfg.Dialect)
		}
		if cfg.MigrationDir() != "migrations/postgres" {
			t.Errorf("expected migration dir migrations/postgres, got %s", cfg.MigrationDir())
		}
	})

	t.Run("resolves supabase target case-insensitively", func(t *testing.T) {
		t.Setenv("SUPABASE_DB_HOST", "db.supabase.co")
		t.Setenv("SUPABASE_DB_PORT", "5432")
		t.Setenv("SUPABASE_DB_USER", "postgres")
		t.Setenv("SUPABASE_DB_PASSWORD", "supapassword")
		t.Setenv("SUPABASE_DB_NAME", "postgres")
		t.Setenv("SUPABASE_DB_SSLMODE", "require")

		cfg := ResolveDBConfig("Supabase")
		if cfg.Host != "db.supabase.co" {
			t.Errorf("expected host db.supabase.co, got %s", cfg.Host)
		}
		expectedDSN := "postgresql://postgres:supapassword@db.supabase.co:5432/postgres?sslmode=require"
		if cfg.DSN == nil || *cfg.DSN != expectedDSN {
			t.Errorf("expected DSN %s, got %v", expectedDSN, cfg.DSN)
		}
		if cfg.Dialect != "postgres" {
			t.Errorf("expected dialect postgres, got %s", cfg.Dialect)
		}
		if cfg.MigrationDir() != "migrations/postgres" {
			t.Errorf("expected migration dir migrations/postgres, got %s", cfg.MigrationDir())
		}
	})

	t.Run("resolves sqlserver target case-insensitively", func(t *testing.T) {
		t.Setenv("SQLSERVER_DB_HOST", "127.0.0.1")
		t.Setenv("SQLSERVER_DB_PORT", "1433")
		t.Setenv("SQLSERVER_DB_USER", "sa")
		t.Setenv("SQLSERVER_DB_PASSWORD", "StrongPassword123!")
		t.Setenv("SQLSERVER_DB_NAME", "komecore")

		cfg := ResolveDBConfig("SQLServer")
		if cfg.Host != "127.0.0.1" {
			t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
		}
		expectedDSN := "sqlserver://sa:StrongPassword123!@127.0.0.1:1433?database=komecore"
		if cfg.DSN == nil || *cfg.DSN != expectedDSN {
			t.Errorf("expected DSN %s, got %v", expectedDSN, cfg.DSN)
		}
		if cfg.Dialect != "sqlserver" {
			t.Errorf("expected dialect sqlserver, got %s", cfg.Dialect)
		}
		if cfg.MigrationDir() != "migrations/sqlserver" {
			t.Errorf("expected migration dir migrations/sqlserver, got %s", cfg.MigrationDir())
		}
	})

	t.Run("defaults to postgres for unknown target", func(t *testing.T) {
		t.Setenv("POSTGRES_DB_HOST", "127.0.0.1")
		t.Setenv("POSTGRES_DB_PORT", "5432")
		t.Setenv("POSTGRES_DB_USER", "komee")
		t.Setenv("POSTGRES_DB_PASSWORD", "password")
		t.Setenv("POSTGRES_DB_NAME", "komecore")
		t.Setenv("POSTGRES_DB_SSLMODE", "disable")

		cfg := ResolveDBConfig("unknown")
		if cfg.Host != "127.0.0.1" {
			t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
		}
		expectedDSN := "postgresql://komee:password@127.0.0.1:5432/komecore?sslmode=disable"
		if cfg.DSN == nil || *cfg.DSN != expectedDSN {
			t.Errorf("expected DSN %s, got %v", expectedDSN, cfg.DSN)
		}
		if cfg.Dialect != "postgres" {
			t.Errorf("expected dialect postgres, got %s", cfg.Dialect)
		}
	})

	t.Run("falls back to DB_TARGET env when target is empty or omitted", func(t *testing.T) {
		t.Setenv("DB_TARGET", "sqlserver")
		t.Setenv("SQLSERVER_DB_HOST", "127.0.0.1")
		t.Setenv("SQLSERVER_DB_PORT", "1433")
		t.Setenv("SQLSERVER_DB_USER", "sa")
		t.Setenv("SQLSERVER_DB_PASSWORD", "StrongPassword123!")
		t.Setenv("SQLSERVER_DB_NAME", "komecore")

		cfg1 := ResolveDBConfig("")
		if cfg1.Dialect != "sqlserver" {
			t.Errorf("expected dialect sqlserver, got %s", cfg1.Dialect)
		}

		cfg2 := ResolveDBConfig()
		if cfg2.Dialect != "sqlserver" {
			t.Errorf("expected dialect sqlserver, got %s", cfg2.Dialect)
		}
	})
}
