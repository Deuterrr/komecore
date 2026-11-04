package authhttp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"komecore/internal/apperror"
	appconfig "komecore/internal/config"
	"komecore/internal/modules/auth/authhttp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_GoogleLogin_Disabled(t *testing.T) {
	// Unconfigured Google OAuth config
	h := authhttp.New(nil, appconfig.GoogleOAuthConfig{
		Enabled: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	w := httptest.NewRecorder()

	err := h.GoogleLogin(w, req)
	require.Error(t, err)

	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok, "expected error to be *apperror.AppError")
	assert.Equal(t, http.StatusNotImplemented, appErr.StatusCode)
	assert.Equal(t, apperror.ErrTypeNotImplemented, appErr.Type)
	assert.Equal(t, apperror.CodeAuthOAuthDisabled, appErr.Code)
	assert.Contains(t, appErr.Message, "google oauth is disabled")
}

func TestHandler_GoogleCallback_Disabled(t *testing.T) {
	// Unconfigured Google OAuth config
	h := authhttp.New(nil, appconfig.GoogleOAuthConfig{
		Enabled: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=xyz&code=abc", nil)
	w := httptest.NewRecorder()

	err := h.GoogleCallback(w, req)
	require.Error(t, err)

	appErr, ok := err.(*apperror.AppError)
	require.True(t, ok, "expected error to be *apperror.AppError")
	assert.Equal(t, http.StatusNotImplemented, appErr.StatusCode)
	assert.Equal(t, apperror.ErrTypeNotImplemented, appErr.Type)
	assert.Equal(t, apperror.CodeAuthOAuthDisabled, appErr.Code)
}

func TestHandler_GoogleLogin_Configured(t *testing.T) {
	// Configured Google OAuth config
	h := authhttp.New(nil, appconfig.GoogleOAuthConfig{
		Enabled:            true,
		ClientID:           "client-123",
		ClientSecret:       "secret-456",
		RedirectURL:        "http://localhost:7129/api/v1/auth/google/callback",
		SuccessRedirectURL: "http://localhost:3000/callback",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
	w := httptest.NewRecorder()

	err := h.GoogleLogin(w, req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.True(t, strings.HasPrefix(loc, "https://accounts.google.com/o/oauth2/auth"))
	assert.Contains(t, loc, "client_id=client-123")

	// State cookie must be set
	cookies := w.Result().Cookies()
	var stateCookieFound bool
	for _, c := range cookies {
		if c.Name == "oauth_state" && c.Value != "" {
			stateCookieFound = true
			break
		}
	}
	assert.True(t, stateCookieFound, "expected oauth_state cookie to be set")
}
