package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type VerifyPasswordResetUsecase struct {
	executor      transaction.Executor
	challengeRepo repository.VerificationChallengeRepository
	pwHasher      repository.PasswordHasher
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewVerifyPasswordResetUsecase(
	executor transaction.Executor,
	challengeRepo repository.VerificationChallengeRepository,
	pwHasher repository.PasswordHasher,
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
		return nil, apperrors.NewNotFound(domain.ErrNotFoundChallenge.Error())
	}

	if challenge.Purpose != domain.OTPPurposePasswordReset {
		return nil, apperrors.NewNotFound(domain.ErrNotFoundChallenge.Error())
	}

	if challenge.ConsumedAt != nil {
		return nil, apperrors.NewConflict(domain.ErrConsumedChallenge.Error())
	}

	if challenge.VerifiedAt != nil {
		return nil, apperrors.NewConflict(domain.ErrVerifiedChallenge.Error())
	}

	if challenge.ExpiresAt.Before(now) {
		return nil, apperrors.NewConflict(domain.ErrExpiredChallenge.Error())
	}

	if challenge.AttemptCount >= 5 {
		return nil, apperrors.NewConflict(domain.ErrMaxAttemptReached.Error())
	}

	if err := u.pwHasher.Compare(challenge.CodeHash, input.OTP); err != nil {
		challenge.AttemptCount++

		if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
			return nil, fmt.Errorf("failed to update challenge attempts: %w", err)
		}

		return nil, apperrors.NewUnauthorized(domain.ErrInvalidOTP.Error())
	}

	challenge.VerifiedAt = &now
	if err := u.challengeRepo.Save(ctx, u.executor, *challenge); err != nil {
		return nil, fmt.Errorf("failed to update challenge verification state: %w", err)
	}

	return &challenge.ID, nil
}
