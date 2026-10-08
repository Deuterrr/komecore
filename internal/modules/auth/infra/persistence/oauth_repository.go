package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OAuthConnectionRepository struct{}

func NewOAuthConnectionRepository() *OAuthConnectionRepository {
	return &OAuthConnectionRepository{}
}

func (r *OAuthConnectionRepository) GetByProviderAndSubject(ctx context.Context, exec transaction.Executor,
	provider domain.OAuthProvider,
	subject string,
) (*domain.OAuthConnection, error) {
	query := `
		SELECT
			id,
			user_id,
			provider,
			subject,
			email,
			last_login_at,
			created_at
		FROM
			oauth_connections
		WHERE
			provider = $1 AND subject = $2 AND deleted_at IS NULL
		LIMIT 1
	`

	var m domain.OAuthConnection
	err := exec.QueryRow(ctx, query, string(provider), subject).Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.Subject,
		&m.Email,
		&m.LastLoginAt,
		&m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query oauth connection by provider and subject failed: %w", err)
	}

	return &m, nil
}

func (r *OAuthConnectionRepository) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*domain.OAuthConnection, error) {
	query := `
		SELECT
			id,
			user_id,
			provider,
			subject,
			email,
			last_login_at,
			created_at
		FROM
			oauth_connections
		WHERE
			user_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var m domain.OAuthConnection
	err := exec.QueryRow(ctx, query, userID).Scan(
		&m.ID,
		&m.UserID,
		&m.Provider,
		&m.Subject,
		&m.Email,
		&m.LastLoginAt,
		&m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query oauth connection by user id failed: %w", err)
	}

	return &m, nil
}

func (r *OAuthConnectionRepository) Create(ctx context.Context, exec transaction.Executor, conn domain.OAuthConnection) error {
	query := `
		INSERT INTO oauth_connections (
			id,
			user_id,
			provider,
			subject,
			email,
			last_login_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := exec.Exec(ctx, query,
		conn.ID,
		conn.UserID,
		string(conn.Provider),
		conn.Subject,
		conn.Email,
		conn.LastLoginAt,
		conn.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert oauth connection failed: %w", err)
	}

	return nil
}

func (r *OAuthConnectionRepository) UpdateLastLogin(ctx context.Context, exec transaction.Executor,
	id uuid.UUID,
	lastLoginAt time.Time,
) error {
	query := `
		UPDATE oauth_connections
		SET
			last_login_at = $2
		WHERE
			id = $1 AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, id, lastLoginAt)
	if err != nil {
		return fmt.Errorf("update oauth last login failed: %w", err)
	}

	return nil
}

func (r *OAuthConnectionRepository) DeleteByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) error {
	query := `
		UPDATE oauth_connections
		SET deleted_at = NOW()
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete oauth connections failed: %w", err)
	}

	return nil
}
