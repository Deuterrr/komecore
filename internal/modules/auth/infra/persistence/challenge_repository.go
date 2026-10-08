package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ChallengeRepository struct{}

func NewChallengeRepository() *ChallengeRepository {
	return &ChallengeRepository{}
}

func (r *ChallengeRepository) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*domain.VerificationChallenge, error) {
	query := `
		SELECT
			id,
			user_id,
			type,
			channel,
			purpose,
			target,
			code_hash,
			expires_at,
			verified_at,
			attempt_count,
			consumed_at,
			created_at
		FROM verification_challenges
		WHERE id = $1
		LIMIT 1
	`

	var vC domain.VerificationChallenge
	err := exec.QueryRow(ctx, query, id).Scan(
		&vC.ID,
		&vC.UserID,
		&vC.Type,
		&vC.Channel,
		&vC.Purpose,
		&vC.Target,
		&vC.CodeHash,
		&vC.ExpiresAt,
		&vC.VerifiedAt,
		&vC.AttemptCount,
		&vC.ConsumedAt,
		&vC.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query account by id failed: %w", err)
	}

	return &vC, nil
}

func (r *ChallengeRepository) Create(ctx context.Context, exec transaction.Executor, challenge domain.VerificationChallenge) error {
	query := `
		INSERT INTO verification_challenges (
			id,
			user_id,
			type,
			channel,
			purpose,
			target,
			code_hash,
			expires_at,
			attempt_count,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`

	_, err := exec.Exec(ctx, query,
		challenge.ID,
		challenge.UserID,
		challenge.Type,
		challenge.Channel,
		challenge.Purpose,
		challenge.Target,
		challenge.CodeHash,
		challenge.ExpiresAt,
		challenge.AttemptCount,
		challenge.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to create verification challenge: %w", err)
	}

	return nil
}

func (r *ChallengeRepository) Save(ctx context.Context, exec transaction.Executor, challenge domain.VerificationChallenge) error {
	query := `
		INSERT INTO verification_challenges (
			id,
			user_id,
			type,
			channel,
			purpose,
			target,
			code_hash,
			expires_at,
			verified_at,
			consumed_at,
			attempt_count,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			type = EXCLUDED.type,
			channel = EXCLUDED.channel,
			purpose = EXCLUDED.purpose,
			target = EXCLUDED.target,
			code_hash = EXCLUDED.code_hash,
			expires_at = EXCLUDED.expires_at,
			verified_at = EXCLUDED.verified_at,
			consumed_at = EXCLUDED.consumed_at,
			attempt_count = EXCLUDED.attempt_count
	`

	_, err := exec.Exec(ctx, query,
		challenge.ID,
		challenge.UserID,
		challenge.Type,
		challenge.Channel,
		challenge.Purpose,
		challenge.Target,
		challenge.CodeHash,
		challenge.ExpiresAt,
		challenge.VerifiedAt,
		challenge.ConsumedAt,
		challenge.AttemptCount,
		challenge.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to save verification challenge: %w", err)
	}

	return nil
}
