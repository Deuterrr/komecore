package config

import (
	"time"
)

// JWTConfig holds authentication token signing keys and session lifetime durations.
type JWTConfig struct {
	Secret string
	Exp    time.Duration
}

// LoadJWTConfig resolves signing keys and token lifespans, enforcing minimum key entropy in production.
func LoadJWTConfig() JWTConfig {
	secret := GetEnv("JWT_SECRET")
	ValidateJWTSecret(secret)

	expStr := GetEnv("JWT_EXPIRATION", "")
	if expStr == "" {
		expStr = GetEnv("JWT_EXP", "24m")
	}

	exp, err := time.ParseDuration(expStr)
	if err != nil || exp <= 0 {
		exp = 24 * time.Minute
	}

	return JWTConfig{
		Secret: secret,
		Exp:    exp,
	}
}

// ValidateJWTSecret validates that the JWT secret satisfies minimum length requirements in production.
func ValidateJWTSecret(secret string) {
	if GetEnv("APP_ENV", "") == "production" && len(secret) < 32 {
		panic("JWT_SECRET must be at least 32 characters long in production")
	}
}
