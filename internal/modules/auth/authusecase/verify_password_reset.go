package authusecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type VerifyPasswordResetUsecase struct {
	executor      transaction.Executor
	challengeRepo authrepo.VerificationChallengeRepository
	pwHasher      authrepo.PasswordHasher
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewVerifyPasswordResetUsecase(
	executor transaction.Executor,
	challengeRepo authrepo.VerificationChallengeRepository,
	pwHasher authrepo.PasswordHasher,
	auditLogger applogger.AuditLogger,
) *VerifyPasswordResetUsecase {
	return &VerifyPasswordResetUsecase{
		executor:      executor,
		challengeRepo: challengeRepo,
		pwHasher:      pwHasher,
		auditLogger:   auditLogger,
	}
}

func (u *VerifyPasswordResetUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type VerifyPasswordResetParams struct {
	ChallengeID uuid.UUID
	OTP         string
}

func (u *VerifyPasswordResetUsecase) Execute(
	ctx context.Context,
	input VerifyPasswordResetParams,
) (verifiedID *uuid.UUID, err error) {
	now := appclock.Now()

	challenge, err := u.challengeRepo.GetByID(ctx, u.executor, input.ChallengeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	if challenge == nil {
		return nil, apperrors.NewNotFound(authdomain.ErrNotFoundChallenge.Error())
	}

	if challenge.Purpose != authdomain.OTPPurposePasswordReset {
		return nil, apperrors.NewNotFound(authdomain.ErrNotFoundChallenge.Error())
	}

	if challenge.ConsumedAt != nil {
		return nil, apperrors.NewConflict(authdomain.ErrConsumedChallenge.Error())
	}

	if challenge.VerifiedAt != nil {
		return nil, apperrors.NewConflict(authdomain.ErrVerifiedChallenge.Error())
	}

	if challenge.ExpiresAt.Before(now) {
		return nil, apperrors.NewConflict(authdomain.ErrExpiredChallenge.Error())
	}

	if challenge.AttemptCount >= 5 {
		return nil, apperrors.NewConflict(authdomain.ErrMaxAttemptReached.Error())
	}

	if err := u.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++

		if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}

		return nil, apperrors.NewUnauthorized(authdomain.ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
		return nil, fmt.Errorf("failed to update challenge verification state: %w", err)
	}

	return &challenge.ID, nil
}
