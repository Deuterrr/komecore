package testdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"komecore/internal/config"
	database "komecore/internal/infra/db"
	"komecore/internal/infra/transactor"
	pgxtransactor "komecore/internal/infra/transactor/pgx"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDB bundles an ephemeral PostgreSQL testcontainer, a live pgxpool.Pool,
// clean architecture transactor/executor, and migration utilities.
type TestDB struct {
	Container  *PostgresContainer
	Pool       *pgxpool.Pool
	DSN        string
	Config     config.DatabaseConfig
	Executor   transactor.Executor
	Transactor transactor.Transactor
	RepoRoot   string
}

// FindRepoRoot traverses upwards from the current directory to find the directory containing go.mod.
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// NewTestDB provisions an ephemeral Postgres 17 container, establishes connection pool,
// and sets up configuration for migrations.
func NewTestDB(ctx context.Context) (*TestDB, error) {
	ctr, err := StartPostgresContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := ctr.ConnectionString(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("get connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("connect to test db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("ping test db: %w", err)
	}

	repoRoot, err := FindRepoRoot()
	if err != nil {
		pool.Close()
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("find repo root: %w", err)
	}

	migPath := filepath.Join(repoRoot, "migrations", "postgres")

	cfg := config.DatabaseConfig{
		Dialect:         "postgres",
		DSN:             &dsn,
		MigrationSource: migPath,
	}

	executor := pgxtransactor.NewPgxExecutor(pool)
	trans := pgxtransactor.NewPgxTransactor(pool)

	return &TestDB{
		Container:  ctr,
		Pool:       pool,
		DSN:        dsn,
		Config:     cfg,
		Executor:   executor,
		Transactor: trans,
		RepoRoot:   repoRoot,
	}, nil
}

// ApplyMigrations runs all UP migrations against the ephemeral database.
func (tdb *TestDB) ApplyMigrations() error {
	return database.RunMigration(tdb.Config)
}

// RollbackOneStep rolls back 1 migration step.
func (tdb *TestDB) RollbackOneStep() error {
	return database.RunRollback(tdb.Config)
}

// RollbackAll rolls back all applied migrations to version 0.
func (tdb *TestDB) RollbackAll() error {
	return database.RunMigrationDown(tdb.Config)
}

// Truncate cleans all tables in the public schema except schema_migrations.
func (tdb *TestDB) Truncate(ctx context.Context) error {
	return TruncateTables(ctx, tdb.Pool)
}

// Close closes the database connection pool and terminates the Docker container.
func (tdb *TestDB) Close(ctx context.Context) error {
	if tdb.Pool != nil {
		tdb.Pool.Close()
	}
	if tdb.Container != nil {
		return tdb.Container.Terminate(ctx)
	}
	return nil
}
