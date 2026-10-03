package config

import (
	"strconv"
	"strings"
)

type AppConfig struct {
	Name               string
	Codename           string // alias to Name for backward compatibility
	Host               string
	Port               string
	Env                string
	CORSAllowedOrigins []string
	ServerTimeout      int  // in seconds
	ShutdownTimeout    int  // in seconds
	AutoMigrate        bool // Auto-run migrations on boot
}

func LoadAppConfig() AppConfig {
	timeout, err := strconv.Atoi(GetEnv("SERVER_TIMEOUT", "30"))
	if err != nil {
		timeout = 30
	}

	shutdownTimeout, err := strconv.Atoi(GetEnv("SHUTDOWN_TIMEOUT", "15"))
	if err != nil {
		shutdownTimeout = 15
	}

	name := GetEnv("APP_NAME", "komecore")
	host := GetEnv("APP_HOST", "0.0.0.0")
	port := GetEnv("APP_PORT", "7129")

	autoMigrate := GetEnv("AUTO_MIGRATE", "false") == "true" || GetEnv("APP_ENV") == "development"

	return AppConfig{
		Name:               name,
		Codename:           name,
		Host:               host,
		Port:               port,
		Env:                GetEnv("APP_ENV"),
		CORSAllowedOrigins: splitCSVEnv("APP_CORS_ALLOWED_ORIGINS"),
		ServerTimeout:      timeout,
		ShutdownTimeout:    shutdownTimeout,
		AutoMigrate:        autoMigrate,
	}
}

func splitCSVEnv(key string) []string {
	raw := GetEnv(key, "")
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		values = append(values, value)
	}

	return values
}
