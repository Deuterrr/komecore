//go:build integration

package integration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"komecore/internal/infra/cache"
	"komecore/internal/testutil/testredis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CachedProduct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Stock int    `json:"stock"`
}

func TestIntegration_Redis_CacheAside_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rctr, err := testredis.StartRedisContainer(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err, "must be able to provision ephemeral redis container")
	defer func() {
		_ = rctr.Terminate(ctx)
	}()

	c := rctr.Cache()
	defer func() {
		_ = rctr.FlushDB(ctx)
	}()

	cacheKey := "product:slug:red-rose-bouquet"

	// 1. Initial lookup results in Cache Miss
	var fetched CachedProduct
	err = c.Get(ctx, cacheKey, &fetched)
	require.Error(t, err)
	assert.True(t, errors.Is(err, cache.ErrCacheMiss), "initial lookup should be cache miss")

	// 2. Cache-Aside: Fetch from DB and populate Redis
	original := CachedProduct{
		ID:    "p001",
		Name:  "Red Rose Bouquet",
		Price: 150000,
		Stock: 25,
	}
	err = c.Set(ctx, cacheKey, original, 5*time.Minute)
	require.NoError(t, err, "setting cache entry must succeed")

	// 3. Subsequent lookup results in Cache Hit
	var hit CachedProduct
	err = c.Get(ctx, cacheKey, &hit)
	require.NoError(t, err, "subsequent lookup should be cache hit")
	assert.Equal(t, original.ID, hit.ID)
	assert.Equal(t, original.Name, hit.Name)
	assert.Equal(t, original.Price, hit.Price)
	assert.Equal(t, original.Stock, hit.Stock)

	// 4. Invalidation: Update product and purge cache key
	err = c.Delete(ctx, cacheKey)
	require.NoError(t, err, "deleting cache key must succeed")

	var afterInvalidation CachedProduct
	err = c.Get(ctx, cacheKey, &afterInvalidation)
	require.Error(t, err)
	assert.True(t, errors.Is(err, cache.ErrCacheMiss), "lookup after invalidation should be cache miss")

	// 5. Pattern-based invalidation
	err = c.Set(ctx, "catalog:category:roses:p1", original, 5*time.Minute)
	require.NoError(t, err)
	err = c.Set(ctx, "catalog:category:roses:p2", original, 5*time.Minute)
	require.NoError(t, err)

	err = c.DeletePattern(ctx, "catalog:category:roses:*")
	require.NoError(t, err, "delete pattern must succeed")

	var checkP1, checkP2 CachedProduct
	assert.True(t, errors.Is(c.Get(ctx, "catalog:category:roses:p1", &checkP1), cache.ErrCacheMiss))
	assert.True(t, errors.Is(c.Get(ctx, "catalog:category:roses:p2", &checkP2), cache.ErrCacheMiss))
}

func TestIntegration_Redis_Concurrent_IdempotencyLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rctr, err := testredis.StartRedisContainer(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err)
	defer func() {
		_ = rctr.Terminate(ctx)
	}()

	client := rctr.Client()
	defer func() {
		_ = rctr.FlushDB(ctx)
	}()

	idempotencyKey := "idemp-order-key-9988"
	lockKey := "idemp:lock:" + idempotencyKey
	respKey := "idemp:resp:" + idempotencyKey

	// Simulate 2 concurrent requests arriving with the same Idempotency-Key
	concurrency := 2
	acquiredCount := 0
	rejectedCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	startSignal := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startSignal

			// Try to acquire distributed lock via SETNX
			acquired, err := client.SetNX(ctx, lockKey, "in_flight", 10*time.Second).Result()
			if err != nil {
				return
			}

			mu.Lock()
			if acquired {
				acquiredCount++
			} else {
				rejectedCount++
			}
			mu.Unlock()
		}()
	}

	// Release both requests simultaneously
	close(startSignal)
	wg.Wait()

	assert.Equal(t, 1, acquiredCount, "exactly 1 request must acquire the idempotency lock")
	assert.Equal(t, 1, rejectedCount, "the duplicate request must be rejected (conflict)")

	// Request 1 completes execution and persists cached response
	mockResponseJSON := `{"order_id":"ord-123","total":250000,"status":"pending"}`
	err = client.Set(ctx, respKey, mockResponseJSON, 24*time.Hour).Err()
	require.NoError(t, err)

	// Release lock
	_ = client.Del(ctx, lockKey).Err()

	// Subsequent request with the same idempotency key hits the cached response
	cachedResult, err := client.Get(ctx, respKey).Result()
	require.NoError(t, err)
	assert.Equal(t, mockResponseJSON, cachedResult, "subsequent request must receive cached response")
}
