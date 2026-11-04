package config_test

import (
	"testing"

	"komecore/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestGoogleOAuthConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.GoogleOAuthConfig
		expected bool
	}{
		{
			name: "fully configured and enabled",
			cfg: config.GoogleOAuthConfig{
				Enabled:      true,
				ClientID:     "client-id-xyz",
				ClientSecret: "secret-abc",
			},
			expected: true,
		},
		{
			name: "disabled explicitly despite credentials present",
			cfg: config.GoogleOAuthConfig{
				Enabled:      false,
				ClientID:     "client-id-xyz",
				ClientSecret: "secret-abc",
			},
			expected: false,
		},
		{
			name: "empty client id",
			cfg: config.GoogleOAuthConfig{
				Enabled:      true,
				ClientID:     "",
				ClientSecret: "secret-abc",
			},
			expected: false,
		},
		{
			name: "whitespace client secret",
			cfg: config.GoogleOAuthConfig{
				Enabled:      true,
				ClientID:     "client-id-xyz",
				ClientSecret: "   ",
			},
			expected: false,
		},
		{
			name:     "zero value struct",
			cfg:      config.GoogleOAuthConfig{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cfg.IsConfigured())
		})
	}
}

func TestLoadGoogleOAuthConfig_OptionalFallback(t *testing.T) {
	t.Run("unconfigured environment does not panic and defaults to disabled", func(t *testing.T) {
		t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "")
		t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "")
		t.Setenv("GOOGLE_OAUTH_ENABLED", "")

		cfg := config.LoadGoogleOAuthConfig()
		assert.False(t, cfg.Enabled)
		assert.False(t, cfg.IsConfigured())
	})

	t.Run("inferred enablement when credentials provided without flag", func(t *testing.T) {
		t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test-client")
		t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "test-secret")
		t.Setenv("GOOGLE_OAUTH_ENABLED", "")

		cfg := config.LoadGoogleOAuthConfig()
		assert.True(t, cfg.Enabled)
		assert.True(t, cfg.IsConfigured())
		assert.Equal(t, "test-client", cfg.ClientID)
		assert.Equal(t, "test-secret", cfg.ClientSecret)
	})

	t.Run("explicit disable flag overrides presence of credentials", func(t *testing.T) {
		t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "test-client")
		t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "test-secret")
		t.Setenv("GOOGLE_OAUTH_ENABLED", "false")

		cfg := config.LoadGoogleOAuthConfig()
		assert.False(t, cfg.Enabled)
		assert.False(t, cfg.IsConfigured())
	})
}
