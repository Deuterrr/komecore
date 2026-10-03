package config

import (
	"strings"
)

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	DSN             *string
	Dialect         string // "postgres" | "sqlserver"
	MigrationSource string // optional custom path or file:// URL override
}

// MigrationDir returns the dialect-specific SQL source directory.
// Used by golang-migrate as: "file://" + cfg.MigrationDir()
func (c DatabaseConfig) MigrationDir() string {
	if c.MigrationSource != "" {
		return c.MigrationSource
	}
	return "migrations/" + c.Dialect
}

func LoadDBConfig(
	Host string,
	Port string,
	User string,
	Password string,
	Name string,
	SSLMode string,
	DSN *string,
) DatabaseConfig {
	return DatabaseConfig{
		Host:     Host,
		Port:     Port,
		User:     User,
		Password: Password,
		Name:     Name,
		SSLMode:  SSLMode,
		DSN:      DSN,
	}
}

// ResolveDBConfig determines and loads the active database configuration based on the target.
// Supports "supabase" for remote Supabase DB, "sqlserver" for SQL Server,
// falling back to "postgres" for local/standalone Postgres.
// If target is omitted or empty, it falls back to the DB_TARGET environment variable.
func ResolveDBConfig(target ...string) DatabaseConfig {
	selected := ""
	if len(target) > 0 {
		selected = target[0]
	}
	if strings.TrimSpace(selected) == "" {
		selected = GetEnv("DB_TARGET", "postgres")
	}

	switch strings.ToLower(strings.TrimSpace(selected)) {
	case "supabase":
		supa := LoadSupabaseConfig()
		return DatabaseConfig{
			Host:     supa.Host,
			Port:     supa.Port,
			User:     supa.User,
			Password: supa.Password,
			Name:     supa.Name,
			SSLMode:  supa.SSLMode,
			DSN:      &supa.DSN,
			Dialect:  "postgres",
		}

	case "sqlserver":
		ss := LoadSQLServerConfig()
		return DatabaseConfig{
			Host:     ss.Host,
			Port:     ss.Port,
			User:     ss.User,
			Password: ss.Password,
			Name:     ss.Name,
			DSN:      &ss.DSN,
			Dialect:  "sqlserver",
		}

	default: // "postgres"
		pg := LoadPostgresConfig()
		return DatabaseConfig{
			Host:     pg.Host,
			Port:     pg.Port,
			User:     pg.User,
			Password: pg.Password,
			Name:     pg.Name,
			SSLMode:  pg.SSLMode,
			DSN:      &pg.DSN,
			Dialect:  "postgres",
		}
	}
}
