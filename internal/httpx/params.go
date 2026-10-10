package httpx

import (
	"net/http"

	"komecore/internal/apperror"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func Param(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func ParamUUID(r *http.Request, key string) (uuid.UUID, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return uuid.Nil, apperror.ErrBadRequest
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ErrBadRequest
	}

	return parsed, nil
}
