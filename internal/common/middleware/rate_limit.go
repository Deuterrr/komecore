package appmiddleware

import (
	"net/http"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"
)

// RateLimit returns a middleware that applies sliding-window rate limiting per client IP.
func RateLimit(limiter applimiter.Limiter, logger applogger.Logger) Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if limiter == nil {
				return next(w, r)
			}

			ip := apphttp.ClientIP(r)
			if !isLocalhost(ip) && !limiter.Allow(ip) {
				if logger != nil {
					logger.Warn(r.Context(), "rate limit exceeded",
						applogger.Field{Key: "ip", Value: ip},
						applogger.Field{Key: "path", Value: r.URL.Path},
					)
				}
				return apperrors.NewTooManyRequests(apperrors.ErrTooManyRequests.Error())
			}

			return next(w, r)
		}
	}
}

func isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1" || ip == "localhost"
}
