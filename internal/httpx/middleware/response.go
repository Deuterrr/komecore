package appmiddleware

import (
	"net/http"

	"komecore/internal/httpx"
)

// ErrResponse is an alias for httpx.ErrResponse for backward compatibility.
type ErrResponse = httpx.ErrResponse

// Response intercepts any returned error, formats it using httpx.WriteError,
// and passes the error along for upstream logging and correlation.
func Response() Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			err := next(w, r)
			if err != nil {
				httpx.WriteError(w, r, err)
				return err
			}
			return nil
		}
	}
}
