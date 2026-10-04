package cache

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type memoryItem struct {
	data      []byte
	expiresAt time.Time
}

// MemoryCache provides a thread-safe in-memory cache implementing the Cache interface.
// Useful for offline environments, unit testing, and embedded fallback.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryCache initializes an in-memory Cache instance.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]memoryItem),
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string, dest any) error {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()

	if !ok {
		return ErrCacheMiss
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return ErrCacheMiss
	}

	switch v := dest.(type) {
	case *string:
		*v = string(item.data)
		return nil
	case *[]byte:
		*v = item.data
		return nil
	default:
		return json.Unmarshal(item.data, dest)
	}
}

func (m *MemoryCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := marshalValue(val)
	if err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.items[key] = memoryItem{
		data:      data,
		expiresAt: expiresAt,
	}
	m.mu.Unlock()

	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}

func (m *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k := range m.items {
		// Use filepath.Match or simple prefix/glob matching
		if matchPattern(pattern, k) {
			delete(m.items, k)
		}
	}
	return nil
}

func (m *MemoryCache) SetNX(ctx context.Context, key string, val any, ttl time.Duration) (bool, error) {
	data, err := marshalValue(val)
	if err != nil {
		return false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[key]
	if ok && (item.expiresAt.IsZero() || time.Now().Before(item.expiresAt)) {
		return false, nil
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.items[key] = memoryItem{
		data:      data,
		expiresAt: expiresAt,
	}
	return true, nil
}

func (m *MemoryCache) Ping(ctx context.Context) error {
	return nil
}

func (m *MemoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]memoryItem)
	return nil
}

// NoopCache implements a no-op Cache where reads always miss and writes do nothing.
type NoopCache struct{}

func NewNoopCache() *NoopCache {
	return &NoopCache{}
}

func (n *NoopCache) Get(ctx context.Context, key string, dest any) error {
	return ErrCacheMiss
}

func (n *NoopCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	return nil
}

func (n *NoopCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *NoopCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (n *NoopCache) SetNX(ctx context.Context, key string, val any, ttl time.Duration) (bool, error) {
	return true, nil
}

func (n *NoopCache) Ping(ctx context.Context) error {
	return nil
}

func (n *NoopCache) Close() error {
	return nil
}

func matchPattern(pattern, str string) bool {
	matched, err := filepath.Match(pattern, str)
	if err == nil && matched {
		return true
	}
	// Fallback for simple wildcards like "prefix*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(str, prefix)
	}
	return false
}
