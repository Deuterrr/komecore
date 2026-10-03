package testdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultPostgresImage = "postgres:17-alpine"
	testDBName           = "komecore_test"
	testDBUser           = "test_user"
	testDBPassword       = "test_password"
)

// PostgresContainer encapsulates a testcontainers PostgreSQL container.
type PostgresContainer struct {
	container *postgres.PostgresContainer
}

// StartPostgresContainer starts an ephemeral PostgreSQL container with Postgres 17 Alpine.
func StartPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx,
		defaultPostgresImage,
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	return &PostgresContainer{
		container: pgContainer,
	}, nil
}

// ConnectionString returns the libpq/pgx-compatible DSN.
func (c *PostgresContainer) ConnectionString(ctx context.Context) (string, error) {
	connStr, err := c.container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", err
	}
	// Normalize if double query prefix occurs
	if strings.Contains(connStr, "??") {
		connStr = strings.Replace(connStr, "??", "?", 1)
	}
	return connStr, nil
}

// Terminate shuts down and removes the container.
func (c *PostgresContainer) Terminate(ctx context.Context) error {
	if c.container == nil {
		return nil
	}
	return c.container.Terminate(ctx)
}

// Host returns the host address of the container.
func (c *PostgresContainer) Host(ctx context.Context) (string, error) {
	return c.container.Host(ctx)
}

// Port returns the mapped port for 5432.
func (c *PostgresContainer) Port(ctx context.Context) (string, error) {
	p, err := c.container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return "", err
	}
	return p.Port(), nil
}
