package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"komecore/internal/httpx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?key=value", nil)
	assert.Equal(t, "value", httpx.Query(req, "key"))
	assert.Empty(t, httpx.Query(req, "nonexistent"))
}

func TestQueryInt(t *testing.T) {
	t.Run("returns zero on absent parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		val, err := httpx.QueryInt(req, "page")
		require.NoError(t, err)
		assert.Equal(t, 0, val)
	})

	t.Run("returns zero on empty parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=", nil)
		val, err := httpx.QueryInt(req, "page")
		require.NoError(t, err)
		assert.Equal(t, 0, val)
	})

	t.Run("returns parsed int on valid parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=42", nil)
		val, err := httpx.QueryInt(req, "page")
		require.NoError(t, err)
		assert.Equal(t, 42, val)
	})

	t.Run("returns error on invalid parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?page=abc", nil)
		_, err := httpx.QueryInt(req, "page")
		require.Error(t, err)
	})
}

func TestQueryUUID(t *testing.T) {
	t.Run("returns nil, nil when parameter is omitted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		parsed, err := httpx.QueryUUID(req, "shop_id")
		require.NoError(t, err)
		assert.Nil(t, parsed)
	})

	t.Run("returns nil, nil when parameter is empty string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?shop_id=", nil)
		parsed, err := httpx.QueryUUID(req, "shop_id")
		require.NoError(t, err)
		assert.Nil(t, parsed)
	})

	t.Run("returns nil, error when parameter is invalid uuid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?shop_id=not-a-valid-uuid", nil)
		parsed, err := httpx.QueryUUID(req, "shop_id")
		require.Error(t, err)
		assert.Nil(t, parsed)
	})

	t.Run("returns parsed uuid pointer when parameter is valid", func(t *testing.T) {
		expectedID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/test?shop_id="+expectedID.String(), nil)
		parsed, err := httpx.QueryUUID(req, "shop_id")
		require.NoError(t, err)
		require.NotNil(t, parsed)
		assert.Equal(t, expectedID, *parsed)
	})
}

func TestQueryBool(t *testing.T) {
	t.Run("returns true for boolean true string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?active=true", nil)
		assert.True(t, httpx.QueryBool(req, "active"))
	})

	t.Run("returns false for boolean false string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?active=false", nil)
		assert.False(t, httpx.QueryBool(req, "active"))
	})

	t.Run("returns false for invalid string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?active=notabool", nil)
		assert.False(t, httpx.QueryBool(req, "active"))
	})

	t.Run("returns false when absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.False(t, httpx.QueryBool(req, "active"))
	})
}

func TestQueryIntDefault(t *testing.T) {
	t.Run("returns fallback when absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.Equal(t, 10, httpx.QueryIntDefault(req, "limit", 10))
	})

	t.Run("returns fallback when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=", nil)
		assert.Equal(t, 10, httpx.QueryIntDefault(req, "limit", 10))
	})

	t.Run("returns fallback when invalid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=invalid", nil)
		assert.Equal(t, 10, httpx.QueryIntDefault(req, "limit", 10))
	})

	t.Run("returns parsed int when valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?limit=25", nil)
		assert.Equal(t, 25, httpx.QueryIntDefault(req, "limit", 10))
	})
}
