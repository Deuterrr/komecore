package authusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	applogger "komecore/pkg/logger"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

type RequestPasswordResetUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	accountRepo  auth.AccountRepository
	challengeSvc auth.ChallengeService
	auditLogger  applogger.AuditLogger
	sysLogger    applogger.Logger
}

func NewRequestPasswordResetUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo auth.AccountRepository,
	challengeRepo auth.VerificationChallengeRepository,
	pwHasher auth.PasswordHasher,
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

func (u *RequestPasswordResetUsecase) SetChallengeService(challengeSvc auth.ChallengeService) {
	u.challengeSvc = challengeSvc
}

func (u *RequestPasswordResetUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type RequestPasswordResetParams struct {
	Email       string
	AccountType auth.AccountType
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
		return nil, apperror.NewForbidden("account type mismatch")
	}
	if account.Status != auth.AccountActive {
		return nil, apperror.NewForbidden(auth.ErrEmailNotVerified.Error())
	}

	chID, err := u.challengeSvc.CreateAndSend(ctx, auth.CreateChallengeParams{
		UserID:   &account.UserID,
		Email:    params.Email,
		Purpose:  auth.OTPPurposePasswordReset,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return chID, nil
}
