package config

import (
	"strconv"
	"strings"
	"time"
)

type StorageConfig struct {
	Provider        string // "supabase" | "gcs" | "noop"
	SignedURLExpiry time.Duration
}

func LoadStorageConfig() StorageConfig {
	provider := strings.ToLower(strings.TrimSpace(GetEnv("STORAGE_PROVIDER", "supabase")))

	expiryStr := strings.TrimSpace(GetEnv("STORAGE_SIGNED_URL_EXPIRY", "900"))
	expiry, err := time.ParseDuration(expiryStr)
	if err != nil {
		if seconds, secErr := strconv.Atoi(expiryStr); secErr == nil && seconds > 0 {
			expiry = time.Duration(seconds) * time.Second
		} else {
			expiry = 15 * time.Minute
		}
	}

	return StorageConfig{
		Provider:        provider,
		SignedURLExpiry: expiry,
	}
}
