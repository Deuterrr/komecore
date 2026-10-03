package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"komecore/internal/infra/transactor"

	"github.com/jackc/pgx/v5"
	_ "github.com/microsoft/go-mssqldb"
)

type sqlResult struct {
	res sql.Result
}

func (r sqlResult) RowsAffected() int64 {
	if r.res == nil {
		return 0
	}
	n, _ := r.res.RowsAffected()
	return n
}

type sqlRow struct {
	row *sql.Row
}

func (r sqlRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w: %w", transactor.ErrNoRows, pgx.ErrNoRows, sql.ErrNoRows)
	}
	return err
}

type sqlRows struct {
	rows *sql.Rows
}

func (r sqlRows) Next() bool {
	if r.rows == nil {
		return false
	}
	return r.rows.Next()
}

func (r sqlRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r sqlRows) Close() {
	if r.rows != nil {
		_ = r.rows.Close()
	}
}

func (r sqlRows) Err() error {
	if r.rows != nil {
		return r.rows.Err()
	}
	return nil
}

// SQLRunner is satisfied by both *sql.DB and *sql.Tx.
type SQLRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var pgParamRegex = regexp.MustCompile(`\$([0-9]+)`)

// convertPgParamsToSQLServer translates Postgres positional placeholders ($1, $2) to SQL Server (@p1, @p2).
func convertPgParamsToSQLServer(query string) string {
	return pgParamRegex.ReplaceAllString(query, "@p$1")
}

// SQLServerExecutor adapts any SQLRunner (*sql.DB, *sql.Tx) to transactor.Executor.
type SQLServerExecutor struct {
	runner SQLRunner
}

func NewSQLServerExecutor(runner SQLRunner) *SQLServerExecutor {
	return &SQLServerExecutor{runner: runner}
}

func (e *SQLServerExecutor) Exec(ctx context.Context, sqlQuery string, args ...any) (transactor.Result, error) {
	translated := convertPgParamsToSQLServer(sqlQuery)
	res, err := e.runner.ExecContext(ctx, translated, args...)
	if err != nil {
		return nil, err
	}
	return sqlResult{res: res}, nil
}

func (e *SQLServerExecutor) Query(ctx context.Context, sqlQuery string, args ...any) (transactor.Rows, error) {
	translated := convertPgParamsToSQLServer(sqlQuery)
	rows, err := e.runner.QueryContext(ctx, translated, args...)
	if err != nil {
		return nil, err
	}
	return sqlRows{rows: rows}, nil
}

func (e *SQLServerExecutor) QueryRow(ctx context.Context, sqlQuery string, args ...any) transactor.Row {
	translated := convertPgParamsToSQLServer(sqlQuery)
	row := e.runner.QueryRowContext(ctx, translated, args...)
	return sqlRow{row: row}
}

type SQLServerTransactor struct {
	db *sql.DB
}

func NewSQLServerTransactor(db *sql.DB) transactor.Transactor {
	return &SQLServerTransactor{db: db}
}

func (t *SQLServerTransactor) WithinTransaction(
	ctx context.Context,
	fn func(transactor.Executor) error,
) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(NewSQLServerExecutor(tx)); err != nil {
		return err
	}

	return tx.Commit()
}
