package config

import "strings"

// GoogleOAuthConfig captures credentials and callback URLs for Google OAuth2 social sign-in.
type GoogleOAuthConfig struct {
	Enabled            bool
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	SuccessRedirectURL string
}

// IsConfigured reports whether Google OAuth is enabled and provisioned with non-empty credentials.
func (c GoogleOAuthConfig) IsConfigured() bool {
	return c.Enabled &&
		strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.ClientSecret) != ""
}

// LoadGoogleOAuthConfig loads Google OAuth configuration with safe optional fallbacks,
// preventing server initialization panics when running without external social identity providers.
func LoadGoogleOAuthConfig() GoogleOAuthConfig {
	clientID := GetEnv("GOOGLE_OAUTH_CLIENT_ID", "")
	clientSecret := GetEnv("GOOGLE_OAUTH_CLIENT_SECRET", "")
	redirectURL := GetEnv("GOOGLE_OAUTH_REDIRECT_URL", "")
	successRedirectURL := GetEnv("GOOGLE_OAUTH_SUCCESS_REDIRECT_URL", "")

	enabledStr := strings.ToLower(strings.TrimSpace(GetEnv("GOOGLE_OAUTH_ENABLED", "")))
	var enabled bool
	if enabledStr != "" {
		enabled = enabledStr == "true" || enabledStr == "1"
	} else {
		// Infer enablement from credential presence to avoid requiring explicit flag in legacy envs.
		enabled = strings.TrimSpace(clientID) != "" && strings.TrimSpace(clientSecret) != ""
	}

	return GoogleOAuthConfig{
		Enabled:            enabled,
		ClientID:           clientID,
		ClientSecret:       clientSecret,
		RedirectURL:        redirectURL,
		SuccessRedirectURL: successRedirectURL,
	}
}
