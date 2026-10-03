package appmiddleware

import (
	"net/http"
	"strings"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
)

type ErrResponse struct {
	Type       apperrors.ErrorType `json:"type"`
	StatusCode int                 `json:"status_code"`
	Message    string              `json:"message"`
}

func Response() Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			err := next(w, r)

			if err != nil {
				appErr := apperrors.Resolve(err)

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

				apphttp.WriteJSON(w, appErr.StatusCode, errRes)

				return err
			}

			return nil
		}
	}
}
