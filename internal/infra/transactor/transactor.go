package transactor

import (
	"context"
	"errors"
)

// ErrNoRows is the driver-agnostic sentinel error returned when a query returns no rows.
var ErrNoRows = errors.New("transactor: no rows in result set")

// Result abstracts the outcome of an Exec command (e.g. rows affected).
type Result interface {
	RowsAffected() int64
}

type simpleResult struct {
	rowsAffected int64
}

func (r simpleResult) RowsAffected() int64 {
	return r.rowsAffected
}

// NewResult creates a basic Result with the given rows affected count.
func NewResult(rowsAffected int64) Result {
	return simpleResult{rowsAffected: rowsAffected}
}

// Rows abstracts multi-row query results across drivers.
type Rows interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}

// Row abstracts single-row scan operations across drivers.
type Row interface {
	Scan(dest ...any) error
}

// Executor abstracts database query execution across drivers (pgx, database/sql, etc.).
//
// Both connection pools and database transactions implement this
// contract, allowing repositories to execute queries without being
// coupled to a specific execution context.
type Executor interface {
	Exec(
		ctx context.Context,
		sql string,
		args ...any,
	) (Result, error)

	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) Row
}

// Transactor provides transactional execution for application services.
//
// It enables use cases to define transaction boundaries while delegating
// transaction lifecycle management to the infrastructure layer.
type Transactor interface {
	WithinTransaction(
		ctx context.Context,
		fn func(Executor) error,
	) error
}

// CollectableRow is an alias for Row used during collection operations.
type CollectableRow = Row

// CollectRows scans all rows into a slice using the provided mapping function.
func CollectRows[T any](rows Rows, fn func(row CollectableRow) (T, error)) ([]T, error) {
	var items []T
	for rows.Next() {
		item, err := fn(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// NoopExecutor provides a stub implementation of Executor for tests.
type NoopExecutor struct{}

func (n *NoopExecutor) Exec(ctx context.Context, sql string, args ...any) (Result, error) {
	return simpleResult{rowsAffected: 1}, nil
}

func (n *NoopExecutor) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return nil, nil
}

func (n *NoopExecutor) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return nil
}

// NoopTransactor provides a stub implementation of Transactor for tests.
type NoopTransactor struct{}

func (n *NoopTransactor) WithinTransaction(ctx context.Context, fn func(Executor) error) error {
	return fn(&NoopExecutor{})
}
