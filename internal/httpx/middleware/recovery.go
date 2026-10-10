package appmiddleware

import (
	"net/http"
	"runtime/debug"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	applogger "komecore/pkg/logger"
)

func Recovery(log applogger.Logger) Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					log.Error(r.Context(), "panic recovered",
						applogger.Field{Key: "panic", Value: rec},
						applogger.Field{Key: "stack", Value: string(stack)},
						applogger.Field{Key: "path", Value: r.URL.Path},
						applogger.Field{Key: "method", Value: r.Method},
					)

					err = apperror.ErrInternal
				}
			}()

			return next(w, r)
		}
	}
}
