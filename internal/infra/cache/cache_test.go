package cache_test

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sampleData struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

func TestMemoryCache_BasicOperations(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache()

	t.Run("Get returns ErrCacheMiss on nonexistent key", func(t *testing.T) {
		var dest string
		err := c.Get(ctx, "nonexistent", &dest)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})

	t.Run("Set and Get string value", func(t *testing.T) {
		err := c.Set(ctx, "greeting", "hello world", 10*time.Minute)
		require.NoError(t, err)

		var val string
		err = c.Get(ctx, "greeting", &val)
		require.NoError(t, err)
		assert.Equal(t, "hello world", val)
	})

	t.Run("Set and Get struct value JSON", func(t *testing.T) {
		item := sampleData{Name: "Gaming Mouse", Price: 499000}
		err := c.Set(ctx, "product:1", item, 10*time.Minute)
		require.NoError(t, err)

		var retrieved sampleData
		err = c.Get(ctx, "product:1", &retrieved)
		require.NoError(t, err)
		assert.Equal(t, item, retrieved)
	})

	t.Run("Delete removes key", func(t *testing.T) {
		err := c.Set(ctx, "temp", "val", 10*time.Minute)
		require.NoError(t, err)

		err = c.Delete(ctx, "temp")
		require.NoError(t, err)

		var dest string
		err = c.Get(ctx, "temp", &dest)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})

	t.Run("DeletePattern removes matching keys", func(t *testing.T) {
		_ = c.Set(ctx, "cache:product:slug:item-1", "data1", 10*time.Minute)
		_ = c.Set(ctx, "cache:product:slug:item-2", "data2", 10*time.Minute)
		_ = c.Set(ctx, "cache:user:profile:123", "data3", 10*time.Minute)

		err := c.DeletePattern(ctx, "cache:product*")
		require.NoError(t, err)

		var dest string
		assert.ErrorIs(t, c.Get(ctx, "cache:product:slug:item-1", &dest), cache.ErrCacheMiss)
		assert.ErrorIs(t, c.Get(ctx, "cache:product:slug:item-2", &dest), cache.ErrCacheMiss)
		assert.NoError(t, c.Get(ctx, "cache:user:profile:123", &dest))
	})

	t.Run("SetNX locks and prevents overwrite while active", func(t *testing.T) {
		acquired, err := c.SetNX(ctx, "lock:order:1", "IN_FLIGHT", 5*time.Minute)
		require.NoError(t, err)
		assert.True(t, acquired)

		// Second call must fail
		second, err := c.SetNX(ctx, "lock:order:1", "IN_FLIGHT", 5*time.Minute)
		require.NoError(t, err)
		assert.False(t, second)
	})

	t.Run("Key expires after TTL", func(t *testing.T) {
		err := c.Set(ctx, "short-lived", "quick", 20*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(30 * time.Millisecond)

		var dest string
		err = c.Get(ctx, "short-lived", &dest)
		assert.ErrorIs(t, err, cache.ErrCacheMiss)
	})
}

func TestNoopCache_Operations(t *testing.T) {
	ctx := context.Background()
	n := cache.NewNoopCache()

	var dest string
	assert.ErrorIs(t, n.Get(ctx, "any", &dest), cache.ErrCacheMiss)
	assert.NoError(t, n.Set(ctx, "any", "val", time.Minute))
	assert.NoError(t, n.Delete(ctx, "any"))
	assert.NoError(t, n.DeletePattern(ctx, "*"))

	ok, err := n.SetNX(ctx, "any", "val", time.Minute)
	assert.NoError(t, err)
	assert.True(t, ok)
}
