package transactor_test

import (
	"errors"
	"testing"

	"komecore/internal/infra/transactor"

	"github.com/jackc/pgx/v5/pgconn"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
)

func TestMapDBError(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, transactor.MapDBError(nil))
	})

	t.Run("PostgreSQL 23505 wraps ErrUniqueViolation", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
		mapped := transactor.MapDBError(pgErr)
		assert.True(t, errors.Is(mapped, transactor.ErrUniqueViolation))
		assert.True(t, errors.Is(mapped, pgErr))
	})

	t.Run("PostgreSQL 23503 wraps ErrForeignKeyViolation", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23503", Message: "foreign key error"}
		mapped := transactor.MapDBError(pgErr)
		assert.True(t, errors.Is(mapped, transactor.ErrForeignKeyViolation))
		assert.True(t, errors.Is(mapped, pgErr))
	})

	t.Run("SQL Server 2601 wraps ErrUniqueViolation", func(t *testing.T) {
		msErr := mssql.Error{Number: 2601, Message: "duplicate key"}
		mapped := transactor.MapDBError(msErr)
		assert.True(t, errors.Is(mapped, transactor.ErrUniqueViolation))
		var extracted mssql.Error
		assert.True(t, errors.As(mapped, &extracted))
		assert.Equal(t, int32(2601), extracted.Number)
	})

	t.Run("SQL Server 2627 pointer wraps ErrUniqueViolation", func(t *testing.T) {
		msErr := &mssql.Error{Number: 2627, Message: "unique constraint"}
		mapped := transactor.MapDBError(msErr)
		assert.True(t, errors.Is(mapped, transactor.ErrUniqueViolation))
		var extracted *mssql.Error
		assert.True(t, errors.As(mapped, &extracted))
		assert.Equal(t, int32(2627), extracted.Number)
	})

	t.Run("SQL Server 547 wraps ErrForeignKeyViolation", func(t *testing.T) {
		msErr := mssql.Error{Number: 547, Message: "foreign key constraint"}
		mapped := transactor.MapDBError(msErr)
		assert.True(t, errors.Is(mapped, transactor.ErrForeignKeyViolation))
		var extracted mssql.Error
		assert.True(t, errors.As(mapped, &extracted))
		assert.Equal(t, int32(547), extracted.Number)
	})

	t.Run("unrecognized error returns original error without wrapping", func(t *testing.T) {
		stdErr := errors.New("timeout error")
		mapped := transactor.MapDBError(stdErr)
		assert.Equal(t, stdErr, mapped)
		assert.False(t, errors.Is(mapped, transactor.ErrUniqueViolation))
		assert.False(t, errors.Is(mapped, transactor.ErrForeignKeyViolation))
	})
}
