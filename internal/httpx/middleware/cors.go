package appmiddleware

import (
	"net/http"
	"strings"

	"komecore/internal/httpx"
)

func CORS(allowedOrigins []string) Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return next(w, r)
			}

			if !isAllowedOrigin(origin, allowedOrigins) {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return nil
				}

				return next(w, r)
			}

			headers := w.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Access-Control-Allow-Credentials", "true")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			headers.Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Requested-With, X-Account-Type")
			headers.Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(w, r)
		}
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return true
		}

		if strings.EqualFold(origin, allowedOrigin) {
			return true
		}
	}

	return false
}
