package appmiddleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	"komecore/internal/common/authctx"
	"komecore/internal/infra/cache"
)

type IdempotencyStatus string

const (
	IdempotencyStatusInFlight  IdempotencyStatus = "IN_FLIGHT"
	IdempotencyStatusCompleted IdempotencyStatus = "COMPLETED"
)

// IdempotencyRecord stores cached request outcome in Redis.
type IdempotencyRecord struct {
	Status     IdempotencyStatus   `json:"status"`
	StatusCode int                 `json:"status_code"`
	Body       []byte              `json:"body"`
	Headers    map[string][]string `json:"headers,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
}

type IdempotencyMiddleware struct {
	cache cache.Cache
}

func NewIdempotencyMiddleware(c cache.Cache) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{cache: c}
}

// RequireIdempotency enforces an Idempotency-Key header on mutating operations.
// It guards against duplicate concurrent execution and replays completed responses on retries.
func (m *IdempotencyMiddleware) RequireIdempotency() Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			key := strings.TrimSpace(apphttp.GetIdempotencyKeyHeader(r))
			if key == "" {
				return apperrors.NewBadRequest("Idempotency-Key header is required")
			}
			if len(key) > 128 {
				return apperrors.NewBadRequest("Idempotency-Key header exceeds maximum length of 128 characters")
			}

			if m == nil || m.cache == nil {
				return next(w, r)
			}

			ctx := r.Context()
			actorID := resolveActorID(ctx, r)
			cacheKey := fmt.Sprintf("idemp:%s:%s", actorID, key)

			// Check existing record
			var record IdempotencyRecord
			err := m.cache.Get(ctx, cacheKey, &record)
			if err == nil {
				if record.Status == IdempotencyStatusInFlight {
					return apperrors.NewConflict("a request with this idempotency key is currently processing")
				}
				if record.Status == IdempotencyStatusCompleted {
					w.Header().Set("X-Cache", "HIT-IDEMPOTENCY")
					if ct := record.Headers["Content-Type"]; len(ct) > 0 {
						w.Header().Set("Content-Type", ct[0])
					} else {
						w.Header().Set("Content-Type", "application/json")
					}
					w.WriteHeader(record.StatusCode)
					_, _ = w.Write(record.Body)
					return nil
				}
			}

			// Lock key with SetNX
			inFlightRecord := IdempotencyRecord{
				Status:    IdempotencyStatusInFlight,
				CreatedAt: time.Now(),
			}
			acquired, err := m.cache.SetNX(ctx, cacheKey, inFlightRecord, 120*time.Second)
			if err != nil {
				return fmt.Errorf("failed to reserve idempotency lock: %w", err)
			}
			if !acquired {
				return apperrors.NewConflict("a request with this idempotency key is currently processing")
			}

			rec := &bodyRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			handlerErr := next(rec, r)
			if handlerErr == nil && rec.statusCode >= 200 && rec.statusCode < 300 {
				headersMap := make(map[string][]string)
				if ct := w.Header().Values("Content-Type"); len(ct) > 0 {
					headersMap["Content-Type"] = ct
				}
				completedRecord := IdempotencyRecord{
					Status:     IdempotencyStatusCompleted,
					StatusCode: rec.statusCode,
					Body:       rec.body.Bytes(),
					Headers:    headersMap,
					CreatedAt:  time.Now(),
				}
				_ = m.cache.Set(ctx, cacheKey, completedRecord, 24*time.Hour)
			} else {
				// Failed or non-2xx request: release lock so client can retry with corrected payload
				_ = m.cache.Delete(ctx, cacheKey)
			}

			return handlerErr
		}
	}
}

type bodyRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (b *bodyRecorder) WriteHeader(code int) {
	b.statusCode = code
	b.ResponseWriter.WriteHeader(code)
}

func (b *bodyRecorder) Write(data []byte) (int, error) {
	b.body.Write(data)
	return b.ResponseWriter.Write(data)
}

func resolveActorID(ctx context.Context, r *http.Request) string {
	if authCtx, ok := authctx.GetAuthContext(ctx); ok && authCtx.IsAuthenticated {
		if authCtx.CustomerID != nil {
			return authCtx.CustomerID.String()
		}
		if authCtx.UserID.String() != "00000000-0000-0000-0000-000000000000" {
			return authCtx.UserID.String()
		}
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	return "anon"
}
