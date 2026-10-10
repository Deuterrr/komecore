package authusecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	applogger "komecore/pkg/logger"
)

type LogoutUsecase struct {
	transactor     transaction.Transactor
	refreshTknRepo authrepo.RefreshTokenRepository
	sessionRepo    authrepo.SessionRepository
	auditLogger    applogger.AuditLogger
}

func NewLogoutUsecase(
	transactor transaction.Transactor,
	refreshTknRepo authrepo.RefreshTokenRepository,
	sessionRepo authrepo.SessionRepository,
	auditLogger applogger.AuditLogger,
) *LogoutUsecase {
	return &LogoutUsecase{
		transactor:     transactor,
		refreshTknRepo: refreshTknRepo,
		sessionRepo:    sessionRepo,
		auditLogger:    auditLogger,
	}
}

func (u *LogoutUsecase) Execute(
	ctx context.Context,
	authCtx authdomain.AuthContext,
) error {
	err := u.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := u.refreshTknRepo.RevokeBySessionID(
				ctx,
				exec,
				authCtx.SessionID,
			); err != nil {
				return fmt.Errorf("failed to revoke refresh token %w", err)
			}

			if err := u.sessionRepo.RevokeByID(
				ctx,
				exec,
				authCtx.SessionID,
			); err != nil {
				return fmt.Errorf("failed to revoke refresh token %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "logout",
		Resource:   "session",
		ResourceID: authCtx.SessionID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"user_id": authCtx.UserID.String()},
	})

	return nil
}
