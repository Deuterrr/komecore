package config

type GoogleOAuthConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	SuccessRedirectURL string
}

func LoadGoogleOAuthConfig() GoogleOAuthConfig {
	return GoogleOAuthConfig{
		ClientID:           GetEnv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret:       GetEnv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:        GetEnv("GOOGLE_OAUTH_REDIRECT_URL"),
		SuccessRedirectURL: GetEnv("GOOGLE_OAUTH_SUCCESS_REDIRECT_URL"),
	}
}
