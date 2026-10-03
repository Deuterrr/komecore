package pgx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"komecore/internal/infra/transactor"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxResult struct {
	tag pgconn.CommandTag
}

func (r pgxResult) RowsAffected() int64 {
	return r.tag.RowsAffected()
}

type pgxRow struct {
	row pgxv5.Row
}

func (r pgxRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, pgxv5.ErrNoRows) {
		return fmt.Errorf("%w: %w: %w", transactor.ErrNoRows, pgxv5.ErrNoRows, sql.ErrNoRows)
	}
	return err
}

type pgxRows struct {
	rows pgxv5.Rows
}

func (r pgxRows) Next() bool {
	return r.rows.Next()
}

func (r pgxRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r pgxRows) Close() {
	r.rows.Close()
}

func (r pgxRows) Err() error {
	return r.rows.Err()
}

// PgxRunner is satisfied by both *pgxpool.Pool and pgxv5.Tx.
type PgxRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row
}

// PgxExecutor adapts any PgxRunner (*pgxpool.Pool, pgxv5.Tx) to transactor.Executor.
type PgxExecutor struct {
	runner PgxRunner
}

func NewPgxExecutor(runner PgxRunner) *PgxExecutor {
	return &PgxExecutor{runner: runner}
}

func (e *PgxExecutor) Exec(ctx context.Context, sql string, args ...any) (transactor.Result, error) {
	tag, err := e.runner.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgxResult{tag: tag}, nil
}

func (e *PgxExecutor) Query(ctx context.Context, sql string, args ...any) (transactor.Rows, error) {
	rows, err := e.runner.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgxRows{rows: rows}, nil
}

func (e *PgxExecutor) QueryRow(ctx context.Context, sql string, args ...any) transactor.Row {
	row := e.runner.QueryRow(ctx, sql, args...)
	return pgxRow{row: row}
}

type PgxTransactor struct {
	db *pgxpool.Pool
}

func NewPgxTransactor(db *pgxpool.Pool) transactor.Transactor {
	return &PgxTransactor{db: db}
}

func (t *PgxTransactor) WithinTransaction(
	ctx context.Context,
	fn func(transactor.Executor) error,
) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(NewPgxExecutor(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
