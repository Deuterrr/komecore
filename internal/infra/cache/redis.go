package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"komecore/internal/config"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

// NewRedisCache initializes a Redis client using the provided configuration.
func NewRedisCache(cfg config.RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", cfg.Addr(), err)
	}

	return &RedisCache{client: client}, nil
}

// NewRedisCacheFromClient wraps an existing redis.Client.
func NewRedisCacheFromClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string, dest any) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return err
	}

	switch v := dest.(type) {
	case *string:
		*v = string(data)
		return nil
	case *[]byte:
		*v = data
		return nil
	default:
		if err := json.Unmarshal(data, dest); err != nil {
			return fmt.Errorf("failed to unmarshal cached data for key %s: %w", key, err)
		}
		return nil
	}
}

func (r *RedisCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	payload, err := marshalValue(val)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, payload, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var keysToDelete []string

	for {
		var keys []string
		var err error
		keys, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys matching %s: %w", pattern, err)
		}

		keysToDelete = append(keysToDelete, keys...)

		if cursor == 0 {
			break
		}
	}

	if len(keysToDelete) > 0 {
		return r.client.Del(ctx, keysToDelete...).Err()
	}

	return nil
}

func (r *RedisCache) SetNX(ctx context.Context, key string, val any, ttl time.Duration) (bool, error) {
	payload, err := marshalValue(val)
	if err != nil {
		return false, err
	}
	return r.client.SetNX(ctx, key, payload, ttl).Result()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func marshalValue(val any) ([]byte, error) {
	switch v := val.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		bytes, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cache value: %w", err)
		}
		return bytes, nil
	}
}
