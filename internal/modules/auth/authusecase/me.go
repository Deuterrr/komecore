package authusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/auth/authsvc"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userrepo"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

const meWriteInterval = 5 * time.Minute

type MeUsecase struct {
	exec        transaction.Executor
	accountRepo authrepo.AccountRepository
	userRepo    userrepo.UserRepository
	oauthRepo   authrepo.OAuthConnectionRepository
	auditLogger applogger.AuditLogger
	sysLogger   applogger.Logger
}

func NewMeUsecase(
	exec transaction.Executor,
	accountRepo authrepo.AccountRepository,
	userRepo userrepo.UserRepository,
	oauthRepo authrepo.OAuthConnectionRepository,
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
	Account authdomain.Account
	Actor   authdomain.Actor
	User    *userdomain.User
	OAuth   *authdomain.OAuthConnection
}

func (u *MeUsecase) Execute(ctx context.Context, authCtx authdomain.AuthContext) (result *MeResult, err error) {
	account, err := u.accountRepo.GetByUserID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return nil, apperror.NewNotFound("account not found")
	}

	if account.Status != authdomain.AccountActive {
		return nil, apperror.NewForbidden(authdomain.ErrEmailNotVerified.Error())
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
		actor = &authdomain.Actor{}
	}

	user, err := u.userRepo.GetByID(ctx, u.exec, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	var oauthConn *authdomain.OAuthConnection
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
