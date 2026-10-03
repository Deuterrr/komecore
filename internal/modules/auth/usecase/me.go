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
	userDomain "komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

const meWriteInterval = 5 * time.Minute

type MeUsecase struct {
	exec        transaction.Executor
	accountRepo repository.AccountRepository
	userRepo    userRepo.UserRepository
	oauthRepo   repository.OAuthConnectionRepository
	auditLogger applogger.AuditLogger
	sysLogger   applogger.Logger
}

func NewMeUsecase(
	exec transaction.Executor,
	accountRepo repository.AccountRepository,
	userRepo userRepo.UserRepository,
	oauthRepo repository.OAuthConnectionRepository,
) *MeUsecase {
	return &MeUsecase{
		exec:        exec,
		accountRepo: accountRepo,
		userRepo:    userRepo,
		oauthRepo:   oauthRepo,
	}
}

func (u *MeUsecase) SetAuditLogger(auditLogger applogger.AuditLogger) {
	u.auditLogger = auditLogger
}

func (u *MeUsecase) SetSysLogger(sysLogger applogger.Logger) {
	u.sysLogger = sysLogger
}

type MeResult struct {
	Account domain.Account
	Actor   domain.Actor
	User    *userDomain.User
	OAuth   *domain.OAuthConnection
}

func (u *MeUsecase) Execute(ctx context.Context, authCtx domain.AuthContext) (result *MeResult, err error) {
	account, err := u.accountRepo.GetByUserID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return nil, apperrors.NewNotFound("account not found")
	}

	if account.Status != domain.AccountActive {
		return nil, apperrors.NewForbidden(domain.ErrEmailNotVerified.Error())
	}

	now := appclock.Now()
	if account.LastLoginAt == nil ||
		now.Sub(*account.LastLoginAt) >= meWriteInterval {
		if err := u.accountRepo.UpdateLastLoginAt(ctx, u.exec,
			account.ID,
			now,
		); err != nil {
			return nil, fmt.Errorf("failed to update last login at: %w", err)
		}
		account.LastLoginAt = &now
	}

	actor := service.ActorFromAuthContext(&authCtx)
	if actor == nil {
		actor = &domain.Actor{}
	}

	user, err := u.userRepo.GetByID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	var oauthConn *domain.OAuthConnection
	if u.oauthRepo != nil {
		oauthConn, err = u.oauthRepo.GetByUserID(ctx, u.exec, authCtx.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve oauth connection: %w", err)
		}
	}

	return &MeResult{
		Account: *account,
		Actor:   *actor,
		User:    user,
		OAuth:   oauthConn,
	}, nil
}
