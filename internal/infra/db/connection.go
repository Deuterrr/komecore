package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"komecore/internal/config"
	"komecore/internal/infra/transactor"
	pgxTransactor "komecore/internal/infra/transactor/pgx"
	sqlserverTransactor "komecore/internal/infra/transactor/sqlserver"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/microsoft/go-mssqldb"
)

type Connection struct {
	Dialect    string
	Pool       *pgxpool.Pool
	SqlDB      *sql.DB
	Executor   transactor.Executor
	Transactor transactor.Transactor
}

// NewPostgresConnection connects to PostgreSQL using pgxpool.
func NewPostgresConnection(dsn string) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET timezone TO 'Asia/Jakarta'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{
		Dialect:    "postgres",
		Pool:       pool,
		Executor:   pgxTransactor.NewPgxExecutor(pool),
		Transactor: pgxTransactor.NewPgxTransactor(pool),
	}, nil
}

// NewSQLServerConnection connects to Microsoft SQL Server using database/sql.
func NewSQLServerConnection(dsn string) (*Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlserver database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlserver database: %w", err)
	}

	return &Connection{
		Dialect:    "sqlserver",
		SqlDB:      db,
		Executor:   sqlserverTransactor.NewSQLServerExecutor(db),
		Transactor: sqlserverTransactor.NewSQLServerTransactor(db),
	}, nil
}

// NewConnection is a convenience constructor delegating based on dialect.
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	if cfg.DSN == nil || *cfg.DSN == "" {
		return nil, fmt.Errorf("database: DSN is not configured")
	}

	if cfg.Dialect == "sqlserver" {
		return NewSQLServerConnection(*cfg.DSN)
	}
	return NewPostgresConnection(*cfg.DSN)
}

func (c *Connection) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
	if c.SqlDB != nil {
		_ = c.SqlDB.Close()
	}
}
