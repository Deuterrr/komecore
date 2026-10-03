package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotEnv attempts to load environment variables from a local .env file.
// If no .env file is found (e.g., in Docker or cloud production environments),
// it silently falls back to system environment variables.
func LoadDotEnv() {
	if err := godotenv.Load(); err == nil {
		log.Println("[INFO] Environment configuration loaded from disk (.env)")
	}
}

// GetEnv retrieves an environment variable or returns a default value / panics.
func GetEnv(key string, defaultKey ...string) string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		if defaultKey != nil {
			return defaultKey[0]
		}

		panic(fmt.Sprintf("Missing required environment variable: %s", key))
	}
	return val
}
