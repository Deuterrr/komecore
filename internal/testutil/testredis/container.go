package testredis

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/infra/cache"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultRedisImage = "redis:7-alpine"
	redisPort         = "6379/tcp"
)

// RedisContainer encapsulates an ephemeral Redis 7 Alpine testcontainer.
type RedisContainer struct {
	container testcontainers.Container
	client    *redis.Client
	cache     cache.Cache
	addr      string
}

// StartRedisContainer starts an ephemeral Redis 7 container and returns a connected handle.
func StartRedisContainer(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        defaultRedisImage,
		ExposedPorts: []string{redisPort},
		WaitingFor:   wait.ForLog("Ready to accept connections tcp").WithStartupTimeout(60 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis host: %w", err)
	}

	port, err := ctr.MappedPort(ctx, redisPort)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("failed to get redis port: %w", err)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisContainer{
		container: ctr,
		client:    client,
		cache:     cache.NewRedisCacheFromClient(client),
		addr:      addr,
	}, nil
}

// Client returns the underlying *redis.Client.
func (c *RedisContainer) Client() *redis.Client {
	return c.client
}

// Cache returns the wrapped cache.Cache implementation.
func (c *RedisContainer) Cache() cache.Cache {
	return c.cache
}

// Addr returns the host:port string of the Redis container.
func (c *RedisContainer) Addr() string {
	return c.addr
}

// FlushDB purges all keys in the active Redis database.
func (c *RedisContainer) FlushDB(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	return c.client.FlushDB(ctx).Err()
}

// Terminate closes the client and terminates the Docker container.
func (c *RedisContainer) Terminate(ctx context.Context) error {
	if c.client != nil {
		_ = c.client.Close()
	}
	if c.container != nil {
		return c.container.Terminate(ctx)
	}
	return nil
}
