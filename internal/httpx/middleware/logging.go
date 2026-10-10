package appmiddleware

import (
	"net/http"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

func Logging(log applogger.Logger) Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			requestID := uuid.NewString()
			ctx := applogger.WithRequestID(r.Context(), requestID)
			ctx = applogger.WithClientIP(ctx, httpx.ClientIP(r))
			r = r.WithContext(ctx)

			w.Header().Set("X-Request-ID", requestID)

			rw := newResponseRecorder(w)

			start := appclock.Now()
			err := next(rw, r)

			// request_id and client_ip are already in ctx and prepended
			// automatically by the Logger implementations.
			fields := []applogger.Field{
				{Key: "layer", Value: applogger.LayerMiddleware},
				{Key: "method", Value: r.Method},
				{Key: "path", Value: r.URL.Path},
				{Key: "status_code", Value: rw.statusCode},
				{Key: "user_agent", Value: r.UserAgent()},
				{Key: "duration_ms", Value: time.Since(start).Milliseconds()},
				{Key: "category", Value: applogger.CategorySystem},
			}

			if err != nil {
				appErr := apperror.Resolve(err)

				fields = append(fields,
					applogger.Field{Key: "error", Value: err.Error()},
					applogger.Field{Key: "type", Value: appErr.Type},
				)

				switch appErr.LogLevel {
				case applogger.LogLevelWarn:
					log.Warn(r.Context(), "request warning", fields...)

				case applogger.LogLevelInfo:
					log.Info(r.Context(), "request info", fields...)

				default:
					log.Error(r.Context(), "request failed", fields...)
				}

			} else {
				warningMsg := rw.Header().Get("X-Warning")
				if warningMsg != "" {
					fields = append(fields, applogger.Field{Key: "warning", Value: warningMsg})
					log.Warn(r.Context(), "request completed with warning", fields...)
				} else if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/health" {
					log.Info(r.Context(), "request success", fields...)
				}
			}

			return err
		}
	}
}
