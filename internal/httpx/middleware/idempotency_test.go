package appmiddleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appmiddleware "komecore/internal/httpx/middleware"
	"komecore/internal/infra/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyMiddleware(t *testing.T) {
	t.Run("missing Idempotency-Key returns 400 Bad Request", func(t *testing.T) {
		c := cache.NewMemoryCache()
		mw := appmiddleware.NewIdempotencyMiddleware(c)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := mw.RequireIdempotency()(func(w http.ResponseWriter, r *http.Request) error {
			handlerCalled = true
			return nil
		})

		err := handler(w, req)
		assert.Error(t, err)
		assert.False(t, handlerCalled)
	})

	t.Run("key exceeding 128 chars returns 400 Bad Request", func(t *testing.T) {
		c := cache.NewMemoryCache()
		mw := appmiddleware.NewIdempotencyMiddleware(c)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		longKey := string(make([]byte, 129))
		req.Header.Set("Idempotency-Key", longKey)
		w := httptest.NewRecorder()

		handler := mw.RequireIdempotency()(func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})

		err := handler(w, req)
		assert.Error(t, err)
	})

	t.Run("first request succeeds and subsequent duplicate request replays cached response", func(t *testing.T) {
		c := cache.NewMemoryCache()
		mw := appmiddleware.NewIdempotencyMiddleware(c)

		callCount := 0
		downstream := func(w http.ResponseWriter, r *http.Request) error {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"order_id":"123","status":"pending"}`))
			return nil
		}

		handler := mw.RequireIdempotency()(downstream)

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		req1.Header.Set("Idempotency-Key", "uuid-test-1")
		w1 := httptest.NewRecorder()

		err := handler(w1, req1)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, http.StatusCreated, w1.Code)
		assert.Equal(t, `{"order_id":"123","status":"pending"}`, w1.Body.String())

		// Second request with same idempotency key
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		req2.Header.Set("Idempotency-Key", "uuid-test-1")
		w2 := httptest.NewRecorder()

		err = handler(w2, req2)
		require.NoError(t, err)
		assert.Equal(t, 1, callCount, "downstream handler must NOT be called on replay")
		assert.Equal(t, http.StatusCreated, w2.Code)
		assert.Equal(t, "HIT-IDEMPOTENCY", w2.Header().Get("X-Cache"))
		assert.Equal(t, `{"order_id":"123","status":"pending"}`, w2.Body.String())
	})

	t.Run("concurrent in-flight request returns 409 Conflict", func(t *testing.T) {
		c := cache.NewMemoryCache()
		mw := appmiddleware.NewIdempotencyMiddleware(c)

		// Pre-populate key with IN_FLIGHT
		_ = c.Set(t.Context(), "idemp:anon:uuid-flight-1", appmiddleware.IdempotencyRecord{
			Status:    appmiddleware.IdempotencyStatusInFlight,
			CreatedAt: time.Now(),
		}, time.Minute)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		req.Header.Set("Idempotency-Key", "uuid-flight-1")
		w := httptest.NewRecorder()

		handler := mw.RequireIdempotency()(func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})

		err := handler(w, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currently processing")
	})

	t.Run("failed request releases lock allowing retry", func(t *testing.T) {
		c := cache.NewMemoryCache()
		mw := appmiddleware.NewIdempotencyMiddleware(c)

		failFirst := true
		handler := mw.RequireIdempotency()(func(w http.ResponseWriter, r *http.Request) error {
			if failFirst {
				failFirst = false
				w.WriteHeader(http.StatusBadRequest)
				return errors.New("invalid stock")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
			return nil
		})

		// 1. First attempt fails
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		req1.Header.Set("Idempotency-Key", "uuid-retry-1")
		w1 := httptest.NewRecorder()

		err := handler(w1, req1)
		assert.Error(t, err)

		// 2. Second attempt with same key succeeds because lock was released
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/order", nil)
		req2.Header.Set("Idempotency-Key", "uuid-retry-1")
		w2 := httptest.NewRecorder()

		err = handler(w2, req2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, `{"success":true}`, w2.Body.String())
	})
}
