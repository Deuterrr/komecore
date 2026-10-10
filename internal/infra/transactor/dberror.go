package transactor

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	mssql "github.com/microsoft/go-mssqldb"
)

// Sentinel database engine errors mapped across drivers (PostgreSQL & SQL Server).
var (
	ErrUniqueViolation     = errors.New("database: unique constraint violation")
	ErrForeignKeyViolation = errors.New("database: foreign key violation")
)

// MapDBError inspects driver-specific errors (PostgreSQL pgconn.PgError and SQL Server mssql.Error)
// using SQLSTATE/error codes and wraps them in database sentinel errors without string matching.
func MapDBError(err error) error {
	if err == nil {
		return nil
	}

	// PostgreSQL error code inspection
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		// unique_violation
		case "23505":
			return fmt.Errorf("%w: %w", ErrUniqueViolation, err)
		// foreign_key_violation
		case "23503":
			return fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
		}
	}

	// SQL Server error code inspection
	var mssqlErr mssql.Error
	var mssqlErrPtr *mssql.Error
	if errors.As(err, &mssqlErr) || errors.As(err, &mssqlErrPtr) {
		num := mssqlErr.Number
		if mssqlErrPtr != nil {
			num = mssqlErrPtr.Number
		}
		switch num {
		// Cannot insert duplicate key / UNIQUE KEY constraint violation
		case 2601, 2627:
			return fmt.Errorf("%w: %w", ErrUniqueViolation, err)
		// FOREIGN KEY constraint violation
		case 547:
			return fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
		}
	}

	return err
}
