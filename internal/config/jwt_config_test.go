package config

import (
	"testing"
	"time"
)

func TestLoadJWTConfig(t *testing.T) {
	t.Run("defaults to 24 minutes when expiration is omitted", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "")
		t.Setenv("JWT_EXP", "")

		cfg := LoadJWTConfig()
		if cfg.Secret != "test_secret_key_for_testing_only" {
			t.Errorf("expected secret to match, got %s", cfg.Secret)
		}
		if cfg.Exp != 24*time.Minute {
			t.Errorf("expected 24m default expiration, got %v", cfg.Exp)
		}
	})

	t.Run("parses valid JWT_EXPIRATION", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "2h")

		cfg := LoadJWTConfig()
		if cfg.Exp != 2*time.Hour {
			t.Errorf("expected 2h expiration, got %v", cfg.Exp)
		}
	})

	t.Run("parses fallback JWT_EXP when JWT_EXPIRATION is omitted", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "")
		t.Setenv("JWT_EXP", "15m")

		cfg := LoadJWTConfig()
		if cfg.Exp != 15*time.Minute {
			t.Errorf("expected 15m expiration, got %v", cfg.Exp)
		}
	})

	t.Run("prioritizes JWT_EXPIRATION over JWT_EXP", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "1h")
		t.Setenv("JWT_EXP", "15m")

		cfg := LoadJWTConfig()
		if cfg.Exp != 1*time.Hour {
			t.Errorf("expected 1h expiration, got %v", cfg.Exp)
		}
	})

	t.Run("falls back to default 24m when JWT_EXPIRATION is malformed", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "malformed_duration")

		cfg := LoadJWTConfig()
		if cfg.Exp != 24*time.Minute {
			t.Errorf("expected 24m fallback, got %v", cfg.Exp)
		}
	})

	t.Run("falls back to default 24m when duration is non-positive", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test_secret_key_for_testing_only")
		t.Setenv("JWT_EXPIRATION", "-5m")

		cfg := LoadJWTConfig()
		if cfg.Exp != 24*time.Minute {
			t.Errorf("expected 24m fallback for negative duration, got %v", cfg.Exp)
		}
	})

	t.Run("panics in production when secret is shorter than 32 characters", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		t.Setenv("JWT_SECRET", "short_secret")

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for short secret in production, got nil")
			}
			msg, ok := r.(string)
			if !ok || msg != "JWT_SECRET must be at least 32 characters long in production" {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()

		_ = LoadJWTConfig()
	})
}

func TestValidateJWTSecret(t *testing.T) {
	t.Run("panics in production when secret is shorter than 32 characters", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for short secret in production, got nil")
			}
			msg, ok := r.(string)
			if !ok || msg != "JWT_SECRET must be at least 32 characters long in production" {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()

		ValidateJWTSecret("short_secret")
	})

	t.Run("allows secret of 32 characters or longer in production", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		validSecret := "12345678901234567890123456789012" // 32 characters

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic for valid secret: %v", r)
			}
		}()

		ValidateJWTSecret(validSecret)
	})

	t.Run("allows short secret in development", func(t *testing.T) {
		t.Setenv("APP_ENV", "development")

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic in development: %v", r)
			}
		}()

		ValidateJWTSecret("short")
	})

	t.Run("allows short secret when APP_ENV is unset", func(t *testing.T) {
		t.Setenv("APP_ENV", "")

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic when APP_ENV unset: %v", r)
			}
		}()

		ValidateJWTSecret("short")
	})
}
