package appcookie

import (
	"errors"
	"net/http"
	"time"

	"komecore/internal/config"
)

func isSecureCookie() bool {
	env := config.GetEnv("APP_ENV", "development")
	secure := config.GetEnv("COOKIE_SECURE", "false")
	return env == "production" || env == "staging" || secure == "true"
}

var (
	// ErrNoCookie is returned when
	// a requested cookie is not present in the HTTP request
	ErrNoCookie = errors.New("http: named cookie not present")
)

type CookieName string

const (
	CookieAccessToken  CookieName = "access_token"
	CookieRefreshToken CookieName = "refresh_token"

	CookieStaffAccessToken  CookieName = "staff_access_token"
	CookieStaffRefreshToken CookieName = "staff_refresh_token"

	CookieOAuthState CookieName = "oauth_state"

	CookieCustomer        = CookieAccessToken
	CookieCustomerRefresh = CookieRefreshToken
	CookieStaff           = CookieStaffAccessToken
	CookieStaffRefresh    = CookieStaffRefreshToken
)

// Bind sets an HTTP cookie on the response writer
// with the given name, value, and expiration time
//
// The cookie is configured as HttpOnly, Secure (in production),
// and uses SameSiteLaxMode by default
func Bind(w http.ResponseWriter, name CookieName, value string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     string(name),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	})
}

// Extract retrieves a cookie value from the
// incoming HTTP request by name
//
// It returns ErrNoCookie if the cookie
// is not present
func Extract(r *http.Request, coukieName CookieName) (string, error) {
	cookie, err := r.Cookie(string(coukieName))
	if err != nil {
		if errors.Is(err, ErrNoCookie) {
			return "", ErrNoCookie
		}
		return "", err
	}

	if cookie.Value == "" {
		return "", ErrNoCookie
	}

	return cookie.Value, nil
}

// Clear removes a cookie from the client by
// setting an expired cookie with the same name
//
// It effectively instructs the browser to
// delete the cookie
func Clear(w http.ResponseWriter, name CookieName) {
	http.SetCookie(w, &http.Cookie{
		Name:     string(name),
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// ClearAll removes all authentication cookies from the client.
//
// This is used during logout to ensure no stale cookies remain
// from any application context.
func ClearAll(w http.ResponseWriter) {
	Clear(w, CookieCustomer)
	Clear(w, CookieCustomerRefresh)
	Clear(w, CookieStaff)
	Clear(w, CookieStaffRefresh)
}

// Exists checks whether a cookie with the
// given name exists in the HTTP request
//
// It returns true if the cookie is present,
// false otherwise
func Exists(r *http.Request, name CookieName) bool {
	_, err := r.Cookie(string(name))
	return err == nil
}
