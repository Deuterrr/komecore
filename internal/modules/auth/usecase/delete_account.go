package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/repository"
	applogger "komecore/pkg/logger"
)

type DeleteAccountUsecase struct {
	transactor          transaction.Transactor
	userDeletionService repository.UserDeletionService
	customerRepo        repository.CustomerRepository
	auditLogger         applogger.AuditLogger
}

func NewDeleteAccountUsecase(
	transactor transaction.Transactor,
	userDeletionService repository.UserDeletionService,
	customerRepo repository.CustomerRepository,
	auditLogger applogger.AuditLogger,
) *DeleteAccountUsecase {
	return &DeleteAccountUsecase{
		transactor:          transactor,
		userDeletionService: userDeletionService,
		customerRepo:        customerRepo,
		auditLogger:         auditLogger,
	}
}

func (u *DeleteAccountUsecase) Execute(ctx context.Context, authCtx domain.AuthContext) error {
	userID := authCtx.UserID

	if err := u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if u.customerRepo != nil && authCtx.CustomerID != nil {
			if err := u.customerRepo.Delete(ctx, exec, *authCtx.CustomerID); err != nil {
				return fmt.Errorf("failed to delete customer record: %w", err)
			}
		}
		if err := u.userDeletionService.DeleteUserRecord(ctx, exec, userID); err != nil {
			return fmt.Errorf("failed to delete user record: %w", err)
		}
		return nil
	}); err != nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "delete_account",
			Resource: "user",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{
				"user_id": userID.String(),
				"error":   err.Error(),
			},
		})
		return err
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category: "user_action",
		Action:   "delete_account",
		Resource: "user",
		Outcome:  applogger.OutcomeSuccess,
		Metadata: map[string]any{
			"user_id": userID.String(),
		},
	})

	return nil
}
