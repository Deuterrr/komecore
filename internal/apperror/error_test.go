package apperror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/infra/transactor"

	"github.com/jackc/pgx/v5/pgconn"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_DefaultCodes(t *testing.T) {
	tests := []struct {
		err          *apperror.AppError
		expectedCode string
		expectedHTTP int
	}{
		{apperror.NewBadRequest("bad request"), apperror.CodeBadRequest, http.StatusBadRequest},
		{apperror.NewInvalidInput("invalid input"), apperror.CodeInvalidInput, http.StatusBadRequest},
		{apperror.NewUnauthorized("unauthorized"), apperror.CodeAuthUnauthorized, http.StatusUnauthorized},
		{apperror.NewForbidden("forbidden"), apperror.CodeAuthForbidden, http.StatusForbidden},
		{apperror.NewNotFound("not found"), apperror.CodeNotFound, http.StatusNotFound},
		{apperror.NewConflict("conflict"), apperror.CodeConflict, http.StatusConflict},
		{apperror.NewMethodNotAllowed("method not allowed"), apperror.CodeMethodNotAllowed, http.StatusMethodNotAllowed},
		{apperror.NewTooManyRequests("too many requests"), apperror.CodeRateLimitExceeded, http.StatusTooManyRequests},
		{apperror.NewInternal(errors.New("db crash")), apperror.CodeInternalServerError, http.StatusInternalServerError},
	}

	for _, tc := range tests {
		resolved := apperror.Resolve(tc.err)
		assert.Equal(t, tc.expectedCode, resolved.Code)
		assert.Equal(t, tc.expectedHTTP, resolved.StatusCode)
		assert.NotNil(t, resolved.Details)
	}
}

func TestAppError_CustomCodesAndBuilders(t *testing.T) {
	err := apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, "product with slug not found")
	assert.Equal(t, apperror.CodeProductNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Equal(t, "product with slug not found", err.Message)

	err.WithDetails("field slug is required", "sku must be valid").WithRequestID("req-123")
	assert.Equal(t, "req-123", err.RequestID)
	require.Len(t, err.Details, 2)
	assert.Equal(t, "field slug is required", err.Details[0])

	rootErr := errors.New("underlying cause")
	err.Wrap(rootErr)
	assert.True(t, errors.Is(err, rootErr))
	assert.Equal(t, rootErr, err.Unwrap())
}

func TestAppError_ResolveDBErrors(t *testing.T) {
	t.Run("resolves mapped transactor.ErrUniqueViolation to Conflict", func(t *testing.T) {
		dbErr := fmt.Errorf("%w: duplicate key value", transactor.ErrUniqueViolation)
		resolved := apperror.Resolve(dbErr)
		assert.Equal(t, apperror.ErrTypeConflict, resolved.Type)
		assert.Equal(t, http.StatusConflict, resolved.StatusCode)
		assert.Equal(t, apperror.CodeConflict, resolved.Code)
	})

	t.Run("resolves mapped transactor.ErrForeignKeyViolation to BadRequest", func(t *testing.T) {
		dbErr := fmt.Errorf("%w: foreign key violation", transactor.ErrForeignKeyViolation)
		resolved := apperror.Resolve(dbErr)
		assert.Equal(t, apperror.ErrTypeBadRequest, resolved.Type)
		assert.Equal(t, http.StatusBadRequest, resolved.StatusCode)
		assert.Equal(t, apperror.CodeBadRequest, resolved.Code)
	})

	t.Run("resolves raw PostgreSQL pgconn.PgError 23505 without string matching", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
		resolved := apperror.Resolve(pgErr)
		assert.Equal(t, apperror.ErrTypeConflict, resolved.Type)
		assert.Equal(t, http.StatusConflict, resolved.StatusCode)
		assert.Equal(t, apperror.CodeConflict, resolved.Code)
	})

	t.Run("resolves raw PostgreSQL pgconn.PgError 23503 without string matching", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23503", Message: "insert or update on table violates foreign key constraint"}
		resolved := apperror.Resolve(pgErr)
		assert.Equal(t, apperror.ErrTypeBadRequest, resolved.Type)
		assert.Equal(t, http.StatusBadRequest, resolved.StatusCode)
		assert.Equal(t, apperror.CodeBadRequest, resolved.Code)
	})

	t.Run("resolves raw SQL Server mssql.Error 2601 to Conflict", func(t *testing.T) {
		msErr := mssql.Error{Number: 2601, Message: "Cannot insert duplicate key"}
		resolved := apperror.Resolve(msErr)
		assert.Equal(t, apperror.ErrTypeConflict, resolved.Type)
		assert.Equal(t, http.StatusConflict, resolved.StatusCode)
		assert.Equal(t, apperror.CodeConflict, resolved.Code)
	})

	t.Run("resolves raw SQL Server mssql.Error 547 to BadRequest", func(t *testing.T) {
		msErr := mssql.Error{Number: 547, Message: "The INSERT statement conflicted with the FOREIGN KEY constraint"}
		resolved := apperror.Resolve(msErr)
		assert.Equal(t, apperror.ErrTypeBadRequest, resolved.Type)
		assert.Equal(t, http.StatusBadRequest, resolved.StatusCode)
		assert.Equal(t, apperror.CodeBadRequest, resolved.Code)
	})

	t.Run("resolves unmapped errors to Internal 500", func(t *testing.T) {
		unmapped := errors.New("connection reset by peer")
		resolved := apperror.Resolve(unmapped)
		assert.Equal(t, apperror.ErrTypeInternal, resolved.Type)
		assert.Equal(t, http.StatusInternalServerError, resolved.StatusCode)
		assert.Equal(t, apperror.CodeInternalServerError, resolved.Code)
	})
}
