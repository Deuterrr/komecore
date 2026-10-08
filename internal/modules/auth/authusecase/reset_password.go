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

type ResetPasswordUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	accountRepo   authrepo.AccountRepository
	sessionRepo   authrepo.SessionRepository
	challengeRepo authrepo.VerificationChallengeRepository
	pwHasher      authrepo.PasswordHasher
	auditLogger   applogger.AuditLogger
	sysLogger     applogger.Logger
}

func NewResetPasswordUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authrepo.AccountRepository,
	sessionRepo authrepo.SessionRepository,
	challengeRepo authrepo.VerificationChallengeRepository,
	pwHasher authrepo.PasswordHasher,
	auditLogger applogger.AuditLogger,
) *ResetPasswordUsecase {
	return &ResetPasswordUsecase{
		executor:      executor,
		transactor:    transactor,
		accountRepo:   accountRepo,
		sessionRepo:   sessionRepo,
		challengeRepo: challengeRepo,
		pwHasher:      pwHasher,
		auditLogger:   auditLogger,
	}
}

func (u *ResetPasswordUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type ResetPasswordParams struct {
	ChallengeID uuid.UUID
	NewPassword string
}

func (u *ResetPasswordUsecase) Execute(ctx context.Context, params ResetPasswordParams) (err error) {
	now := appclock.Now()

	challenge, err := u.challengeRepo.GetByID(ctx, u.executor, params.ChallengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge: %w", err)
	}
	if challenge == nil {
		return apperrors.NewNotFound(authdomain.ErrNotFoundChallenge.Error())
	}
	if challenge.Purpose != authdomain.OTPPurposePasswordReset {
		return apperrors.NewNotFound(authdomain.ErrNotFoundChallenge.Error())
	}
	if challenge.ConsumedAt != nil {
		return apperrors.NewConflict(authdomain.ErrConsumedChallenge.Error())
	}
	if challenge.VerifiedAt == nil {
		return apperrors.NewConflict("challenge is not verified")
	}
	if challenge.ExpiresAt.Before(now) {
		return apperrors.NewConflict(authdomain.ErrExpiredChallenge.Error())
	}
	if challenge.UserID == nil {
		return apperrors.NewInternal(fmt.Errorf("challenge has no user_id bound"))
	}

	hashedPassword, err := u.pwHasher.Hash(params.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	challenge.ConsumedAt = &now

	if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.accountRepo.UpdatePasswordByUserID(ctx, exec, *challenge.UserID, hashedPassword); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
		if err := u.sessionRepo.RevokeAllByUserID(ctx, exec, *challenge.UserID); err != nil {
			return fmt.Errorf("failed to revoke sessions: %w", err)
		}
		if err := u.challengeRepo.Save(ctx, exec, *challenge); err != nil {
			return fmt.Errorf("failed to consume challenge: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
