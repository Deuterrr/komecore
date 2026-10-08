package authusecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/auth/authrepo"
	applogger "komecore/pkg/logger"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

type RequestPasswordResetUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	accountRepo  authrepo.AccountRepository
	challengeSvc authrepo.ChallengeService
	auditLogger  applogger.AuditLogger
	sysLogger    applogger.Logger
}

func NewRequestPasswordResetUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authrepo.AccountRepository,
	challengeRepo authrepo.VerificationChallengeRepository,
	pwHasher authrepo.PasswordHasher,
	otpGen otp.Generator,
	mailer mailer.Sender,
	auditLogger applogger.AuditLogger,
) *RequestPasswordResetUsecase {
	challengeSvc := authsvc.NewChallengeService(
		transactor,
		challengeRepo,
		pwHasher,
		otpGen,
		mailer,
	)

	return &RequestPasswordResetUsecase{
		executor:     executor,
		transactor:   transactor,
		accountRepo:  accountRepo,
		challengeSvc: challengeSvc,
		auditLogger:  auditLogger,
	}
}

func (u *RequestPasswordResetUsecase) SetChallengeService(challengeSvc authrepo.ChallengeService) {
	u.challengeSvc = challengeSvc
}

func (u *RequestPasswordResetUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type RequestPasswordResetParams struct {
	Email       string
	AccountType authdomain.AccountType
}

func (u *RequestPasswordResetUsecase) Execute(ctx context.Context, params RequestPasswordResetParams) (challengeID *uuid.UUID, err error) {
	account, err := u.accountRepo.GetByEmail(ctx, u.executor, params.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		// Silent success for non-existent accounts to prevent email enumeration
		return nil, nil
	}
	if account.Type != params.AccountType {
		return nil, apperrors.NewForbidden("account type mismatch")
	}
	if account.Status != authdomain.AccountActive {
		return nil, apperrors.NewForbidden(authdomain.ErrEmailNotVerified.Error())
	}

	chID, err := u.challengeSvc.CreateAndSend(ctx, authrepo.CreateChallengeParams{
		UserID:   &account.UserID,
		Email:    params.Email,
		Purpose:  authdomain.OTPPurposePasswordReset,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return chID, nil
}
