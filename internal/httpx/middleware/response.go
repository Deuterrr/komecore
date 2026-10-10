package appmiddleware

import (
	"net/http"
	"strings"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
)

type ErrResponse struct {
	Type       apperror.ErrorType `json:"type"`
	StatusCode int                `json:"status_code"`
	Message    string             `json:"message"`
}

func Response() Middleware {
	return func(next httpx.AppHandler) httpx.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			err := next(w, r)

			if err != nil {
				appErr := apperror.Resolve(err)

				msg := err.Error()
				parts := strings.Split(msg, ":")
				if len(parts) > 0 {
					msg = strings.TrimSpace(parts[0])
				}

				errRes := ErrResponse{
					Type:       appErr.Type,
					Message:    msg,
					StatusCode: appErr.StatusCode,
				}

				httpx.WriteJSON(w, appErr.StatusCode, errRes)

				return err
			}

			return nil
		}
	}
}
