package httpx

import (
	"encoding/json"
	"net/http"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/pagination"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

// PaginatedResponse defines the canonical JSON envelope for all paginated collection endpoints.
type PaginatedResponse[T any] struct {
	Items      []T                       `json:"items"`
	Pagination pagination.PaginationMeta `json:"pagination"`
}

// NewPaginatedResponse initializes a PaginatedResponse ensuring items is never serialized as null.
func NewPaginatedResponse[T any](items []T, meta pagination.PaginationMeta) PaginatedResponse[T] {
	if items == nil {
		items = make([]T, 0)
	}
	return PaginatedResponse[T]{
		Items:      items,
		Pagination: meta,
	}
}

// ToUTCPtr converts a time pointer to UTC while preserving nil.
func ToUTCPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

// ErrResponse represents the standardized JSON error envelope for 4xx and 5xx responses.
type ErrResponse struct {
	Type       apperror.ErrorType `json:"type"`
	StatusCode int                `json:"status_code"`
	Message    string             `json:"message"`
	Code       string             `json:"code"`
	RequestID  string             `json:"request_id"`
	Details    []string           `json:"details"`
}

// WriteJSON writes the given data as a JSON response with the specified status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError serializes a standardized error envelope to http.ResponseWriter.
// It resolves the error using apperror.Resolve, ensures request_id correlation,
// and strictly masks 500 internal errors from exposing internal exceptions or query details.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	appErr := apperror.Resolve(err)

	requestID := ""
	if r != nil {
		requestID = applogger.RequestIDFromContext(r.Context())
		if requestID == "" {
			requestID = r.Header.Get("X-Request-ID")
		}
	}
	if requestID == "" {
		requestID = w.Header().Get("X-Request-ID")
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}
	w.Header().Set("X-Request-ID", requestID)

	code := appErr.Code
	if code == "" {
		code = apperror.DefaultCodeForType(appErr.Type)
	}

	msg := appErr.Message
	if appErr.StatusCode >= http.StatusInternalServerError {
		msg = "internal server error"
		code = apperror.CodeInternalServerError
	}

	details := appErr.Details
	if details == nil {
		details = []string{}
	}

	errRes := ErrResponse{
		Type:       appErr.Type,
		StatusCode: appErr.StatusCode,
		Message:    msg,
		Code:       code,
		RequestID:  requestID,
		Details:    details,
	}

	WriteJSON(w, appErr.StatusCode, errRes)
}
