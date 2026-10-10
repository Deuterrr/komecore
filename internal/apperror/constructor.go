package apperror

import applogger "komecore/pkg/logger"

func newAppError(
	t ErrorType,
	msg string,
	statusCode int,
	level applogger.LogLevel,
	err error,
) *AppError {
	return &AppError{
		Type:       t,
		Message:    msg,
		StatusCode: statusCode,
		Code:       DefaultCodeForType(t),
		Details:    []string{},
		LogLevel:   level,
		Err:        err,
	}
}

func newAppErrorWithCode(
	t ErrorType,
	code string,
	msg string,
	statusCode int,
	level applogger.LogLevel,
	err error,
) *AppError {
	if code == "" {
		code = DefaultCodeForType(t)
	}
	return &AppError{
		Type:       t,
		Message:    msg,
		StatusCode: statusCode,
		Code:       code,
		Details:    []string{},
		LogLevel:   level,
		Err:        err,
	}
}

func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

func (e *AppError) WithDetails(details ...string) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithRequestID(reqID string) *AppError {
	e.RequestID = reqID
	return e
}

func (e *AppError) Wrap(err error) *AppError {
	e.Err = err
	return e
}

func NewBadRequest(msg string) *AppError {
	return newAppError(ErrTypeBadRequest, msg, 400, applogger.LogLevelWarn, nil)
}

func NewBadRequestWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeBadRequest, code, msg, 400, applogger.LogLevelWarn, nil)
}

func NewInvalidInput(msg string) *AppError {
	return newAppError(ErrTypeInvalidInput, msg, 400, applogger.LogLevelWarn, nil)
}

func NewInvalidInputWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeInvalidInput, code, msg, 400, applogger.LogLevelWarn, nil)
}

func NewUnauthorized(msg string) *AppError {
	return newAppError(ErrTypeUnauthorized, msg, 401, applogger.LogLevelWarn, nil)
}

func NewUnauthorizedWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeUnauthorized, code, msg, 401, applogger.LogLevelWarn, nil)
}

func NewForbidden(msg string) *AppError {
	return newAppError(ErrTypeForbidden, msg, 403, applogger.LogLevelWarn, nil)
}

func NewForbiddenWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeForbidden, code, msg, 403, applogger.LogLevelWarn, nil)
}

func NewNotFound(msg string) *AppError {
	return newAppError(ErrTypeNotFound, msg, 404, applogger.LogLevelWarn, nil)
}

func NewNotFoundWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeNotFound, code, msg, 404, applogger.LogLevelWarn, nil)
}

func NewConflict(msg string) *AppError {
	return newAppError(ErrTypeConflict, msg, 409, applogger.LogLevelWarn, nil)
}

func NewConflictWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeConflict, code, msg, 409, applogger.LogLevelWarn, nil)
}

func NewMethodNotAllowed(msg string) *AppError {
	return newAppError(ErrTypeMethodNotAllowed, msg, 405, applogger.LogLevelWarn, nil)
}

func NewMethodNotAllowedWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeMethodNotAllowed, code, msg, 405, applogger.LogLevelWarn, nil)
}

func NewTooManyRequests(msg string) *AppError {
	return newAppError(ErrTypeTooManyRequests, msg, 429, applogger.LogLevelWarn, nil)
}

func NewTooManyRequestsWithCode(code, msg string) *AppError {
	return newAppErrorWithCode(ErrTypeTooManyRequests, code, msg, 429, applogger.LogLevelWarn, nil)
}

func NewInternal(err error) *AppError {
	return newAppError(ErrTypeInternal, "internal server error", 500, applogger.LogLevelError, err)
}
