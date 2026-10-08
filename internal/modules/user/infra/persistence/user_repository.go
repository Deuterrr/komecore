package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/user/domain"
	"komecore/internal/modules/user/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.User, error) {
	query := `
		SELECT
			u.id,
			u.name,
			u.username,
			u.phone,
			u.avatar_url,
			u.created_at,
			u.updated_at,
			a.last_login_at,
			COALESCE(a.type::text, 'customer') AS role
		FROM users u
		LEFT JOIN accounts a ON a.user_id = u.id
		WHERE
			u.id = $1
			AND u.deleted_at IS NULL
		LIMIT 1
	`

	var m domain.User

	err := exec.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.Name,
		&m.Username,
		&m.Phone,
		&m.AvatarURL,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.LastLoginAt,
		&m.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return &domain.User{}, fmt.Errorf("query user by id failed: %w", err)
	}

	return &m, nil
}

func (r *UserRepository) GetByUsername(
	ctx context.Context,
	exec transaction.Executor,
	username string,
) (*domain.User, error) {
	query := `
		SELECT
			u.id,
			u.name,
			u.username,
			u.phone,
			u.avatar_url,
			u.created_at,
			u.updated_at,
			a.last_login_at,
			COALESCE(a.type::text, 'customer') AS role
		FROM users u
		LEFT JOIN accounts a ON a.user_id = u.id
		WHERE
			u.username = $1
			AND u.deleted_at IS NULL
		LIMIT 1
	`

	var m domain.User
	err := exec.QueryRow(ctx, query, username).Scan(
		&m.ID,
		&m.Name,
		&m.Username,
		&m.Phone,
		&m.AvatarURL,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.LastLoginAt,
		&m.Role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("query user by username failed: %w", err)
	}

	return &m, nil
}

func (r *UserRepository) CreateUser(
	ctx context.Context,
	exec transaction.Executor,
	props repository.CreateUserProps,
) error {
	query := `
		INSERT INTO users (
			id,
			name,
			username,
			phone,
			avatar_url,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := exec.Exec(ctx, query,
		props.ID,
		props.Name,
		props.Username,
		props.Phone,
		props.AvatarURL,
		props.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user failed: %w", err)
	}

	return nil
}

func (r *UserRepository) SaveProfile(
	ctx context.Context,
	exec transaction.Executor,
	props repository.SaveProfileProps,
) error {
	query := `
		UPDATE users
		SET
			name       = COALESCE($2, name),
			phone      = COALESCE($3, phone),
			avatar_url = COALESCE($4, avatar_url),
			username   = COALESCE($5, username),
			updated_at = $6
		WHERE id = $1;
	`

	_, err := exec.Exec(ctx, query,
		props.UserID,
		props.Name,
		props.Phone,
		props.AvatarURL,
		props.Username,
		props.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save user profile failed: %w", err)
	}

	return nil
}

func (r *UserRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE users
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	res, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("query to delete user failed: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}
