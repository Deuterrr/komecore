package authusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/user"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

const meWriteInterval = 5 * time.Minute

type MeUsecase struct {
	exec        transaction.Executor
	accountRepo auth.AccountRepository
	userRepo    user.Repository
	oauthRepo   auth.OAuthConnectionRepository
	auditLogger applogger.AuditLogger
	sysLogger   applogger.Logger
}

func NewMeUsecase(
	exec transaction.Executor,
	accountRepo auth.AccountRepository,
	userRepo user.Repository,
	oauthRepo auth.OAuthConnectionRepository,
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
	Account auth.Account
	Actor   auth.Actor
	User    *user.User
	OAuth   *auth.OAuthConnection
}

func (u *MeUsecase) Execute(ctx context.Context, authCtx auth.AuthContext) (result *MeResult, err error) {
	account, err := u.accountRepo.GetByUserID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return nil, apperror.NewNotFound("account not found")
	}

	if account.Status != auth.AccountActive {
		return nil, apperror.NewForbidden(auth.ErrEmailNotVerified.Error())
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

	actor := authsvc.ActorFromAuthContext(&authCtx)
	if actor == nil {
		actor = &auth.Actor{}
	}

	user, err := u.userRepo.GetByID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	var oauthConn *auth.OAuthConnection
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
