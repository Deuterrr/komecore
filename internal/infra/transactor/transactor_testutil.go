package transactor

import "context"

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
