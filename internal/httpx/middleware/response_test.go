package appmiddleware_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/httpx"
	appmiddleware "komecore/internal/httpx/middleware"
	applogger "komecore/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseMiddleware_ErrorEnvelope(t *testing.T) {
	t.Run("4xx error returns full envelope with custom code and untruncated message", func(t *testing.T) {
		mw := appmiddleware.Response()
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			// Wrapped error with colons that previously would have been destroyed by strings.Split
			wrapped := fmt.Errorf("outer context: nested detail: %w", errors.New("root error"))
			return apperror.NewBadRequestWithCode(apperror.CodeInvalidInput, wrapped.Error()).
				WithDetails("field email: invalid format")
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Request-ID", "req-test-uuid-400")
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		assert.Error(t, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, "req-test-uuid-400", rec.Header().Get("X-Request-ID"))

		var resp httpx.ErrResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, apperror.ErrTypeBadRequest, resp.Type)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		// Message must NOT be truncated to "outer context"
		assert.Equal(t, "outer context: nested detail: root error", resp.Message)
		assert.Equal(t, apperror.CodeInvalidInput, resp.Code)
		assert.Equal(t, "req-test-uuid-400", resp.RequestID)
		require.Len(t, resp.Details, 1)
		assert.Equal(t, "field email: invalid format", resp.Details[0])
	})

	t.Run("5xx internal error strictly masks database error message", func(t *testing.T) {
		mw := appmiddleware.Response()
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			// Sensitive internal SQL query error
			return fmt.Errorf("pq: syntax error at or near SELECT * FROM secret_credentials WHERE token = 'xyz'")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := applogger.WithRequestID(req.Context(), "req-test-uuid-500")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		assert.Error(t, err)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var resp httpx.ErrResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, apperror.ErrTypeInternal, resp.Type)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		// Internal exception and SQL queries must never leak to client
		assert.Equal(t, "internal server error", resp.Message)
		assert.NotContains(t, resp.Message, "secret_credentials")
		assert.Equal(t, apperror.CodeInternalServerError, resp.Code)
		assert.Equal(t, "req-test-uuid-500", resp.RequestID)
		// Details must be empty array []
		assert.Equal(t, []string{}, resp.Details)
		assert.NotContains(t, rec.Body.String(), `"details":null`)
	})

	t.Run("default error code is populated when not explicitly provided", func(t *testing.T) {
		mw := appmiddleware.Response()
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			return apperror.NewNotFound("item not found")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		_ = handler(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp httpx.ErrResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, apperror.CodeNotFound, resp.Code)
		assert.NotEmpty(t, resp.RequestID)
	})
}
