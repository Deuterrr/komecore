package apperrors

import applogger "komecore/pkg/logger"

func newAppError(
	t ErrorType,
	msg string,
	code int,
	level applogger.LogLevel,
	err error,
) *AppError {
	return &AppError{
		Type:       t,
		Message:    msg,
		StatusCode: code,
		LogLevel:   level,
		Err:        err,
	}
}

func NewBadRequest(msg string) *AppError {
	return newAppError(ErrTypeBadRequest, msg, 400, applogger.LogLevelWarn, nil)
}

func NewInvalidInput(msg string) *AppError {
	return newAppError(ErrTypeInvalidInput, msg, 400, applogger.LogLevelWarn, nil)
}

func NewUnauthorized(msg string) *AppError {
	return newAppError(ErrTypeUnauthorized, msg, 401, applogger.LogLevelWarn, nil)
}

func NewForbidden(msg string) *AppError {
	return newAppError(ErrTypeForbidden, msg, 403, applogger.LogLevelWarn, nil)
}

func NewNotFound(msg string) *AppError {
	return newAppError(ErrTypeNotFound, msg, 404, applogger.LogLevelWarn, nil)
}

func NewConflict(msg string) *AppError {
	return newAppError(ErrTypeConflict, msg, 409, applogger.LogLevelWarn, nil)
}

func NewMethodNotAllowed(msg string) *AppError {
	return newAppError(ErrTypeMethodNotAllowed, msg, 405, applogger.LogLevelWarn, nil)
}

func NewTooManyRequests(msg string) *AppError {
	return newAppError(ErrTypeTooManyRequests, msg, 429, applogger.LogLevelWarn, nil)
}

func NewInternal(err error) *AppError {
	return newAppError(ErrTypeInternal, "internal server error", 500, applogger.LogLevelError, err)
}
