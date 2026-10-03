package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/infra/service"
	"komecore/internal/modules/auth/repository"
	applogger "komecore/pkg/logger"
	mailer "komecore/pkg/mailer"
	otp "komecore/pkg/otp"

	"github.com/google/uuid"
)

type RequestPasswordResetUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	accountRepo  repository.AccountRepository
	challengeSvc repository.ChallengeService
	auditLogger  applogger.AuditLogger
	sysLogger    applogger.Logger
}

func NewRequestPasswordResetUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo repository.AccountRepository,
	challengeRepo repository.VerificationChallengeRepository,
	pwHasher repository.PasswordHasher,
	otpGen otp.Generator,
	mailer mailer.Sender,
	auditLogger applogger.AuditLogger,
) *RequestPasswordResetUsecase {
	challengeSvc := service.NewChallengeService(
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

func (u *RequestPasswordResetUsecase) SetChallengeService(challengeSvc repository.ChallengeService) {
	u.challengeSvc = challengeSvc
}

func (u *RequestPasswordResetUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type RequestPasswordResetParams struct {
	Email       string
	AccountType domain.AccountType
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
	if account.Status != domain.AccountActive {
		return nil, apperrors.NewForbidden(domain.ErrEmailNotVerified.Error())
	}

	chID, err := u.challengeSvc.CreateAndSend(ctx, repository.CreateChallengeParams{
		UserID:   &account.UserID,
		Email:    params.Email,
		Purpose:  domain.OTPPurposePasswordReset,
		Duration: 15 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return chID, nil
}
