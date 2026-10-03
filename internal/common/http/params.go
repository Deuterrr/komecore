package apphttp

import (
	"net/http"

	apperrors "komecore/internal/common/errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func Param(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func ParamUUID(r *http.Request, key string) (uuid.UUID, error) {
	value := chi.URLParam(r, key)
	if value == "" {
		return uuid.Nil, apperrors.ErrBadRequest
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperrors.ErrBadRequest
	}

	return parsed, nil
}
