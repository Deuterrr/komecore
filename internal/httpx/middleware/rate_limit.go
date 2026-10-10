package appmiddleware

import (
	"net/http"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"
)

// RateLimit returns a middleware that applies sliding-window rate limiting per client IP.
func RateLimit(limiter applimiter.Limiter, logger applogger.Logger) Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if limiter == nil {
				return next(w, r)
			}

			ip := httpx.ClientIP(r)
			if !isLocalhost(ip) && !limiter.Allow(ip) {
				if logger != nil {
					logger.Warn(r.Context(), "rate limit exceeded",
						applogger.Field{Key: "ip", Value: ip},
						applogger.Field{Key: "path", Value: r.URL.Path},
					)
				}
				return apperror.NewTooManyRequests(apperror.ErrTooManyRequests.Error())
			}

			return next(w, r)
		}
	}
}

func isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}
