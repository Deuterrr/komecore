package authpersistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RefreshTokenRepository struct{}

func NewRefreshTokenRepository() *RefreshTokenRepository {
	return &RefreshTokenRepository{}
}

func (r *RefreshTokenRepository) GetBySessionID(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) (*authdomain.RefreshToken, error) {
	query := `
		SELECT
			id,
			session_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		FROM
			refresh_tokens
		WHERE
			session_id = $1
		LIMIT 1
	`

	var tkn authdomain.RefreshToken
	err := exec.QueryRow(ctx, query, sessionID).Scan(
		&tkn.ID,
		&tkn.SessionID,
		&tkn.TokenHash,
		&tkn.ExpiresAt,
		&tkn.RevokedAt,
		&tkn.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query to get refresh token by session id failed: %w", err)
	}

	return &tkn, nil
}

func (r *RefreshTokenRepository) RevokeBySessionID(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) error {
	query := `
		UPDATE
			refresh_tokens
		SET
			revoked_at = now()
		WHERE
			session_id = $1
			AND revoked_at IS NULL
	`

	_, err := exec.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("query to revoke refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) Save(ctx context.Context, exec transaction.Executor, refreshToken authdomain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (
			id,
			session_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		)
		VALUES
			($1,$2,$3,$4,$5,$6)
		ON CONFLICT
			(id)
		DO UPDATE SET
			session_id = EXCLUDED.session_id,
			token_hash = EXCLUDED.token_hash,
			expires_at = EXCLUDED.expires_at,
			revoked_at = EXCLUDED.revoked_at
	`

	_, err := exec.Exec(ctx, query,
		refreshToken.ID,
		refreshToken.SessionID,
		refreshToken.TokenHash,
		refreshToken.ExpiresAt,
		refreshToken.RevokedAt,
		refreshToken.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to save refresh token: %w", err)
	}

	return nil
}
