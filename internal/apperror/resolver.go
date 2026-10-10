package apperror

import (
	"errors"

	"komecore/internal/infra/transactor"

	"github.com/jackc/pgx/v5/pgconn"
	mssql "github.com/microsoft/go-mssqldb"
)

// ErrorMapper allows domain modules to register custom sentinel error mappings.
type ErrorMapper func(err error) (*AppError, bool)

var customMappers []ErrorMapper

// RegisterErrorMapper adds a custom domain error mapper to the resolver pipeline.
func RegisterErrorMapper(m ErrorMapper) {
	if m != nil {
		customMappers = append(customMappers, m)
	}
}

// Resolve translates any Go error into a typed, code-aware AppError.
func Resolve(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Code == "" {
			appErr.Code = DefaultCodeForType(appErr.Type)
		}
		if appErr.Details == nil {
			appErr.Details = []string{}
		}
		return appErr
	}

	// Check registered domain error mappers
	for _, mapper := range customMappers {
		if mapped, ok := mapper(err); ok && mapped != nil {
			if mapped.Code == "" {
				mapped.Code = DefaultCodeForType(mapped.Type)
			}
			if mapped.Details == nil {
				mapped.Details = []string{}
			}
			return mapped
		}
	}

	// Check driver-agnostic transactor sentinels via errors.Is
	if errors.Is(err, transactor.ErrUniqueViolation) {
		return NewConflictWithCode(CodeConflict, "resource already exists")
	}
	if errors.Is(err, transactor.ErrForeignKeyViolation) {
		return NewBadRequestWithCode(CodeBadRequest, "referenced resource does not exist or violates foreign key constraint")
	}

	// Direct inspection of PostgreSQL errors via SQLSTATE
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return NewConflictWithCode(CodeConflict, "resource already exists")
		case "23503": // foreign_key_violation
			return NewBadRequestWithCode(CodeBadRequest, "referenced resource does not exist or violates foreign key constraint")
		}
	}

	// Direct inspection of SQL Server errors via driver error code
	var mssqlErr mssql.Error
	var mssqlErrPtr *mssql.Error
	if errors.As(err, &mssqlErr) || errors.As(err, &mssqlErrPtr) {
		num := mssqlErr.Number
		if mssqlErrPtr != nil {
			num = mssqlErrPtr.Number
		}
		switch num {
		case 2601, 2627:
			return NewConflictWithCode(CodeConflict, "resource already exists")
		case 547:
			return NewBadRequestWithCode(CodeBadRequest, "referenced resource does not exist or violates foreign key constraint")
		}
	}

	return NewInternal(err)
}
