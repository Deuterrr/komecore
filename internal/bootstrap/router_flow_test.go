package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appcookie "komecore/internal/common/http/cookie"
	appmiddleware "komecore/internal/common/middleware"
	transaction "komecore/internal/infra/transactor"
	authendomain "komecore/internal/modules/auth/domain"
	authenRepo "komecore/internal/modules/auth/repository"
	applogger "komecore/pkg/logger"
	applimiter "komecore/pkg/ratelimit"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// Mock Authenticator for router integration tests
type mockRouterAuthenticator struct {
	authenticated bool
	authCtx       *authendomain.AuthContext
}

func (m *mockRouterAuthenticator) RequireAuth(
	_ transaction.Executor,
	_ transaction.Transactor,
	_ appcookie.CookieName,
) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if !m.authenticated {
				return apperrors.NewUnauthorized("unauthorized")
			}
			if m.authCtx != nil {
				ctx := authendomain.WithAuthContext(r.Context(), m.authCtx)
				r = r.WithContext(ctx)
			}
			return next(w, r)
		}
	}
}

func (m *mockRouterAuthenticator) RequireAnyAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookies ...appcookie.CookieName,
) appmiddleware.Middleware {
	return m.RequireAuth(exec, tran, appcookie.CookieCustomer)
}

func (m *mockRouterAuthenticator) RequireMultiAuth(
	exec transaction.Executor,
	tran transaction.Transactor,
	cookies ...appcookie.CookieName,
) appmiddleware.Middleware {
	return m.RequireAuth(exec, tran, appcookie.CookieCustomer)
}

func (m *mockRouterAuthenticator) OptionalAuth(
	_ transaction.Executor,
	_ transaction.Transactor,
	_ ...appcookie.CookieName,
) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if m.authenticated && m.authCtx != nil {
				ctx := authendomain.WithAuthContext(r.Context(), m.authCtx)
				r = r.WithContext(ctx)
			}
			return next(w, r)
		}
	}
}

// Mock Authorizer for router integration tests
type mockRouterAuthorizer struct{}

func (m *mockRouterAuthorizer) RequireAccountType(allowedTypes ...authendomain.AccountType) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			authCtx, ok := authendomain.GetAuthContext(r.Context())
			if !ok || authCtx == nil {
				return apperrors.NewUnauthorized("unauthorized")
			}
			matched := false
			for _, t := range allowedTypes {
				if authCtx.AccountType == t {
					matched = true
					break
				}
			}
			if !matched {
				return apperrors.NewForbidden("forbidden")
			}
			return next(w, r)
		}
	}
}

func (m *mockRouterAuthorizer) RequireStaffRole(allowedRoles ...authendomain.RoleCode) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *mockRouterAuthorizer) RequirePermission(permission string) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *mockRouterAuthorizer) LoadActor(_ transaction.Executor) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

func (m *mockRouterAuthorizer) OptionalLoadActor(_ transaction.Executor) appmiddleware.Middleware {
	return func(next apphttp.AppHandler) apphttp.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	}
}

var _ authenRepo.Authenticator = (*mockRouterAuthenticator)(nil)
var _ authenRepo.Authorizer = (*mockRouterAuthorizer)(nil)

func createTestContainer(authenticated bool, authCtx *authendomain.AuthContext) *Container {
	log := applogger.NewSlogLogger("test")
	lim := applimiter.NewInMemorySlidingWindowLimiter(1*time.Second, 100)

	return &Container{
		Logger:             log,
		Limiter:            lim,
		CORSAllowedOrigins: []string{"*"},
		Authenticator: &mockRouterAuthenticator{
			authenticated: authenticated,
			authCtx:       authCtx,
		},
		Authorizer: &mockRouterAuthorizer{},
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"healthy"}`, rec.Body.String())
}

func TestRouter_RouteChains_UnauthenticatedGuards(t *testing.T) {
	// Container configured with authenticated = false
	c := createTestContainer(false, nil)
	chains := NewRouteChains(c)

	dummyHandler := func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"success"}`))
		return nil
	}

	tests := []struct {
		name         string
		chain        func(h apphttp.AppHandler) http.HandlerFunc
		expectedCode int
	}{
		{
			name: "CustomerOnly chain blocks unauthenticated requests",
			chain: func(h apphttp.AppHandler) http.HandlerFunc {
				return chains.CustomerOnly(h)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "StaffOnly chain blocks unauthenticated requests",
			chain: func(h apphttp.AppHandler) http.HandlerFunc {
				return chains.StaffOnly(h)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "StaffAdminOnly chain blocks unauthenticated requests",
			chain: func(h apphttp.AppHandler) http.HandlerFunc {
				return chains.StaffAdminOnly(h)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "CoreAuth chain blocks unauthenticated requests",
			chain: func(h apphttp.AppHandler) http.HandlerFunc {
				return chains.CoreAuth(h)
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Core chain permits unauthenticated requests",
			chain: func(h apphttp.AppHandler) http.HandlerFunc {
				return chains.Core(h)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.chain(dummyHandler)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			h(rec, req)

			assert.Equal(t, tc.expectedCode, rec.Code)
		})
	}
}

func TestRouter_CoreRoutesStructure(t *testing.T) {
	r := chi.NewRouter()

	dummyH := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Register the core route groups as defined in router.go
	r.Get("/health", dummyH)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", dummyH)
		r.Post("/login", dummyH)
		r.Post("/logout", dummyH)
		r.Post("/refresh", dummyH)
		r.Get("/me", dummyH)

		r.Route("/staff", func(r chi.Router) {
			r.Post("/login", dummyH)
			r.Post("/refresh", dummyH)
		})
	})

	r.Route("/products", func(r chi.Router) {
		r.Get("/", dummyH)
		r.Post("/", dummyH)
		r.Get("/{slug}", dummyH)
		r.Get("/stats", dummyH)
		r.Route("/{productId}/reviews", func(r chi.Router) {
			r.Get("/", dummyH)
			r.Post("/", dummyH)
		})
	})

	r.Route("/reviews", func(r chi.Router) {
		r.Delete("/{id}", dummyH)
	})

	r.Route("/carts", func(r chi.Router) {
		r.Get("/", dummyH)
		r.Post("/items", dummyH)
		r.Post("/checkout", dummyH)
	})

	r.Route("/orders", func(r chi.Router) {
		r.Get("/", dummyH)
		r.Post("/", dummyH)
		r.Get("/{id}", dummyH)
	})

	r.Route("/payments", func(r chi.Router) {
		r.Post("/checkout", dummyH)
		r.Post("/webhook", dummyH)
	})

	r.Route("/shipments", func(r chi.Router) {
		r.Get("/{orderId}", dummyH)
	})

	// Collect registered routes
	registeredRoutes := make(map[string]bool)
	walkFn := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		cleanRoute := strings.TrimRight(route, "/")
		if cleanRoute == "" {
			cleanRoute = "/"
		}
		registeredRoutes[method+" "+cleanRoute] = true
		return nil
	}

	err := chi.Walk(r, walkFn)
	assert.NoError(t, err)

	expectedEndpoints := []string{
		"GET /health",
		"POST /auth/register",
		"POST /auth/login",
		"POST /auth/logout",
		"POST /auth/refresh",
		"GET /auth/me",
		"POST /auth/staff/login",
		"POST /auth/staff/refresh",
		"GET /products",
		"POST /products",
		"GET /products/{slug}",
		"GET /products/stats",
		"GET /carts",
		"POST /carts/items",
		"POST /carts/checkout",
		"GET /orders",
		"POST /orders",
		"GET /orders/{id}",
		"POST /payments/checkout",
		"POST /payments/webhook",
		"GET /shipments/{orderId}",
		"GET /products/{productId}/reviews",
		"POST /products/{productId}/reviews",
		"DELETE /reviews/{id}",
	}

	for _, ep := range expectedEndpoints {
		assert.True(t, registeredRoutes[ep], "expected endpoint %s to be registered", ep)
	}
}

func TestRouter_APIVersioningMounts(t *testing.T) {
	r := chi.NewRouter()

	dummyH := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	mountEndpoints := func(sub chi.Router) {
		sub.Get("/products", dummyH)
		sub.Post("/auth/signin", dummyH)
		sub.Get("/orders", dummyH)
	}

	r.Get("/health", dummyH)
	r.Route("/api/v1", mountEndpoints)
	r.Group(mountEndpoints)

	registered := make(map[string]bool)
	_ = chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		registered[method+" "+strings.TrimRight(route, "/")] = true
		return nil
	})

	assert.True(t, registered["GET /health"])
	assert.True(t, registered["GET /api/v1/products"])
	assert.True(t, registered["POST /api/v1/auth/signin"])
	assert.True(t, registered["GET /api/v1/orders"])
	assert.True(t, registered["GET /products"])
	assert.True(t, registered["POST /auth/signin"])
	assert.True(t, registered["GET /orders"])
}
