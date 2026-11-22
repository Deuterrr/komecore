package appmiddleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"komecore/internal/apperror"
	appmiddleware "komecore/internal/httpx/middleware"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyLogger struct {
	lastRequestID string
	lastClientIP  string
	lastMsg       string
	lastFields    []applogger.Field
}

func (l *spyLogger) record(ctx context.Context, msg string, fields ...applogger.Field) {
	l.lastRequestID = applogger.RequestIDFromContext(ctx)
	l.lastClientIP = applogger.ClientIPFromContext(ctx)
	l.lastMsg = msg
	l.lastFields = fields
}

func (l *spyLogger) Info(ctx context.Context, msg string, fields ...applogger.Field) {
	l.record(ctx, msg, fields...)
}

func (l *spyLogger) Error(ctx context.Context, msg string, fields ...applogger.Field) {
	l.record(ctx, msg, fields...)
}

func (l *spyLogger) Warn(ctx context.Context, msg string, fields ...applogger.Field) {
	l.record(ctx, msg, fields...)
}

func (l *spyLogger) Debug(ctx context.Context, msg string, fields ...applogger.Field) {
	l.record(ctx, msg, fields...)
}

func (l *spyLogger) With(_ ...applogger.Field) applogger.Logger {
	return l
}

func TestLoggingMiddleware_RequestID(t *testing.T) {
	t.Run("preserves custom incoming X-Request-ID header", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-Request-ID", "gateway-req-98765")
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		assert.Equal(t, "gateway-req-98765", ctxRequestID)
		assert.Equal(t, "gateway-req-98765", rec.Header().Get("X-Request-ID"))
		assert.Equal(t, "gateway-req-98765", log.lastRequestID)
	})

	t.Run("trims surrounding whitespace from incoming X-Request-ID", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-Request-ID", "  gateway-req-98765 \t ")
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		assert.Equal(t, "gateway-req-98765", ctxRequestID)
		assert.Equal(t, "gateway-req-98765", rec.Header().Get("X-Request-ID"))
		assert.Equal(t, "gateway-req-98765", log.lastRequestID)
	})

	t.Run("generates valid UUID v4 when X-Request-ID is missing", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		require.NotEmpty(t, ctxRequestID)
		parsed, parseErr := uuid.Parse(ctxRequestID)
		require.NoError(t, parseErr)
		assert.Equal(t, uuid.Version(4), parsed.Version())

		assert.Equal(t, ctxRequestID, rec.Header().Get("X-Request-ID"))
		assert.Equal(t, ctxRequestID, log.lastRequestID)
	})

	t.Run("generates fallback UUID v4 when X-Request-ID contains only whitespace", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-Request-ID", "   \t  ")
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		require.NotEmpty(t, ctxRequestID)
		parsed, parseErr := uuid.Parse(ctxRequestID)
		require.NoError(t, parseErr)
		assert.Equal(t, uuid.Version(4), parsed.Version())

		assert.Equal(t, ctxRequestID, rec.Header().Get("X-Request-ID"))
		assert.Equal(t, ctxRequestID, log.lastRequestID)
	})

	t.Run("rejects control characters and falls back to generated UUID v4", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header["X-Request-ID"] = []string{"req-123\x00injected"}
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		parsed, parseErr := uuid.Parse(ctxRequestID)
		require.NoError(t, parseErr)
		assert.Equal(t, uuid.Version(4), parsed.Version())
		assert.Equal(t, ctxRequestID, rec.Header().Get("X-Request-ID"))
	})

	t.Run("truncates X-Request-ID exceeding 128 characters", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		var ctxRequestID string
		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			ctxRequestID = applogger.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
			return nil
		})

		longID := strings.Repeat("a", 160)
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		req.Header.Set("X-Request-ID", longID)
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.NoError(t, err)

		expected := strings.Repeat("a", 128)
		assert.Len(t, ctxRequestID, 128)
		assert.Equal(t, expected, ctxRequestID)
		assert.Equal(t, expected, rec.Header().Get("X-Request-ID"))
		assert.Equal(t, expected, log.lastRequestID)
	})

	t.Run("response header matches ID recorded in log context on error", func(t *testing.T) {
		log := &spyLogger{}
		mw := appmiddleware.Logging(log)

		handler := mw(func(w http.ResponseWriter, r *http.Request) error {
			return apperror.NewBadRequest("invalid payload")
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		req.Header.Set("X-Request-ID", "gateway-err-42")
		rec := httptest.NewRecorder()

		err := handler(rec, req)
		require.Error(t, err)

		assert.Equal(t, "gateway-err-42", rec.Header().Get("X-Request-ID"))
		assert.Equal(t, "gateway-err-42", log.lastRequestID)
	})
}
