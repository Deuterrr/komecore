package apphttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apphttp "komecore/internal/common/http"
	"komecore/internal/common/authctx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth(t *testing.T) {
	t.Run("missing auth context returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx, err := apphttp.RequireAuth(req)
		assert.Nil(t, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication required")
	})

	t.Run("unauthenticated context returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			IsAuthenticated: false,
		}))
		ctx, err := apphttp.RequireAuth(req)
		assert.Nil(t, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication required")
	})

	t.Run("authenticated context returns auth context", func(t *testing.T) {
		userID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			UserID:          userID,
			IsAuthenticated: true,
		}))
		ctx, err := apphttp.RequireAuth(req)
		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.Equal(t, userID, ctx.UserID)
	})
}

func TestRequireCustomer(t *testing.T) {
	t.Run("unauthenticated returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx, custID, err := apphttp.RequireCustomer(req)
		assert.Nil(t, ctx)
		assert.Equal(t, uuid.Nil, custID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication required")
	})

	t.Run("authenticated without customer id returns forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			UserID:          uuid.New(),
			IsAuthenticated: true,
			CustomerID:      nil,
		}))
		ctx, custID, err := apphttp.RequireCustomer(req)
		assert.Nil(t, ctx)
		assert.Equal(t, uuid.Nil, custID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "customer account required")
	})

	t.Run("valid customer returns auth context and customer id", func(t *testing.T) {
		expectedID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			UserID:          uuid.New(),
			IsAuthenticated: true,
			CustomerID:      &expectedID,
		}))
		ctx, custID, err := apphttp.RequireCustomer(req)
		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.Equal(t, expectedID, custID)
	})
}

func TestRequireStaff(t *testing.T) {
	t.Run("unauthenticated returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx, staffID, err := apphttp.RequireStaff(req)
		assert.Nil(t, ctx)
		assert.Equal(t, uuid.Nil, staffID)
		require.Error(t, err)
	})

	t.Run("authenticated without staff id returns forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			UserID:          uuid.New(),
			IsAuthenticated: true,
			StaffID:         nil,
		}))
		ctx, staffID, err := apphttp.RequireStaff(req)
		assert.Nil(t, ctx)
		assert.Equal(t, uuid.Nil, staffID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "staff account required")
	})

	t.Run("valid staff returns auth context and staff id", func(t *testing.T) {
		expectedID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = req.WithContext(authctx.WithAuthContext(req.Context(), &authctx.AuthContext{
			UserID:          uuid.New(),
			IsAuthenticated: true,
			StaffID:         &expectedID,
		}))
		ctx, staffID, err := apphttp.RequireStaff(req)
		require.NoError(t, err)
		require.NotNil(t, ctx)
		assert.Equal(t, expectedID, staffID)
	})
}
