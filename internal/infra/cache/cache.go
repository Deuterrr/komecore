package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss indicates that the requested key does not exist in the cache.
	ErrCacheMiss = errors.New("cache: key not found")
)

// Cache defines the contract for vendor-agnostic distributed and in-memory caching.
type Cache interface {
	// Get retrieves a cached value and deserializes it into dest.
	// Returns ErrCacheMiss if the key does not exist or has expired.
	Get(ctx context.Context, key string, dest any) error

	// Set stores a value in the cache with the specified TTL.
	Set(ctx context.Context, key string, val any, ttl time.Duration) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error

	// DeletePattern scans and removes all keys matching the glob pattern (e.g., "cache:product*").
	DeletePattern(ctx context.Context, pattern string) error

	// SetNX atomically sets the key if it does not already exist (mutex/lock reservation).
	// Returns true if the key was set, false if the key already existed.
	SetNX(ctx context.Context, key string, val any, ttl time.Duration) (bool, error)

	// Ping checks connectivity to the cache store.
	Ping(ctx context.Context) error

	// Close terminates the client connection pool.
	Close() error
}
