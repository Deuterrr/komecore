package authpersistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AccountRepository struct{}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{}
}

func (r *AccountRepository) GetByEmail(
	ctx context.Context,
	exec transaction.Executor,
	email string,
) (*authdomain.Account, error) {
	query := `
		SELECT
			id,
			user_id,
			email,
			password,
			status,
			type,
			last_login_at,
			created_at,
			updated_at
		FROM
			accounts
		WHERE
			email = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var m authdomain.Account
	err := exec.QueryRow(ctx, query, email).Scan(
		&m.ID,
		&m.UserID,
		&m.Email,
		&m.Password,
		&m.Status,
		&m.Type,
		&m.LastLoginAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query account by email failed: %w", err)
	}

	return &m, nil
}

func (r *AccountRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*authdomain.Account, error) {
	query := `
		SELECT
			id,
			user_id,
			email,
			password,
			status,
			type,
			last_login_at,
			created_at,
			updated_at
		FROM
			accounts
		WHERE
			id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var m authdomain.Account
	err := exec.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.UserID,
		&m.Email,
		&m.Password,
		&m.Status,
		&m.Type,
		&m.LastLoginAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query account by id failed: %w", err)
	}

	return &m, nil
}

func (r *AccountRepository) GetByUserID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*authdomain.Account, error) {
	query := `
		SELECT
			id,
			user_id,
			email,
			password,
			status,
			type,
			last_login_at,
			created_at,
			updated_at
		FROM
			accounts
		WHERE
			user_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var m authdomain.Account
	err := exec.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.UserID,
		&m.Email,
		&m.Password,
		&m.Status,
		&m.Type,
		&m.LastLoginAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query account by user id failed: %w", err)
	}

	return &m, nil
}

func (r *AccountRepository) ActivateByUserID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE accounts
		SET
			status = 'active'
		WHERE user_id = $1
	`

	result, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf(
			"query to activate account: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return apperror.NewNotFound(
			authdomain.ErrNotFoundAccount.Error(),
		)
	}

	return nil
}

func (r *AccountRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	acc authdomain.Account,
) error {
	query := `
		INSERT INTO accounts (
			id,
			user_id,
			email,
			password,
			status,
			type,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`

	_, err := exec.Exec(ctx, query,
		acc.ID,
		acc.UserID,
		acc.Email,
		acc.Password,
		acc.Status,
		acc.Type,
		acc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert account failed: %w", err)
	}

	return nil
}

func (r *AccountRepository) UpdatePasswordByUserID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	hashedPassword string,
) error {
	query := `
		UPDATE accounts
		SET
			password = $1,
			updated_at = NOW()
		WHERE user_id = $2
	`

	result, err := exec.Exec(ctx, query, hashedPassword, id)
	if err != nil {
		return fmt.Errorf("query to update account password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperror.NewNotFound(authdomain.ErrNotFoundAccount.Error())
	}

	return nil
}

func (r *AccountRepository) DeleteByUserID(
	ctx context.Context,
	exec transaction.Executor,
	userID uuid.UUID,
) error {
	query := `
		UPDATE accounts
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	res, err := exec.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("query to delete account by user id: %w", err)
	}

	if res.RowsAffected() == 0 {
		return apperror.NewNotFound("account not found or already deleted")
	}

	return nil
}

func (r *AccountRepository) UpdateLastLoginAt(
	ctx context.Context,
	exec transaction.Executor,
	accID uuid.UUID,
	lastLoginAt time.Time,
) error {
	query := `
		UPDATE accounts
		SET
			last_login_at = $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`

	res, err := exec.Exec(ctx, query, lastLoginAt, accID)
	if err != nil {
		return fmt.Errorf("query to update account last login at: %w", err)
	}

	if res.RowsAffected() == 0 {
		return apperror.NewNotFound(authdomain.ErrNotFoundAccount.Error())
	}

	return nil
}
