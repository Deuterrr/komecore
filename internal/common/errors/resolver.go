package apperrors

import (
	"errors"
	"strings"
)

func Resolve(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	if strings.Contains(err.Error(), "duplicate key") {
		return NewConflict("resource already exists")
	}

	return NewInternal(err)
}
