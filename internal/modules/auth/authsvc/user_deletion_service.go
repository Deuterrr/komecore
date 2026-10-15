package authsvc

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth"
	"komecore/internal/modules/user"

	"github.com/google/uuid"
)

type userDeletionServiceImpl struct {
	accountRepo auth.AccountRepository
	oauthRepo   auth.OAuthConnectionRepository
	sessionRepo auth.SessionRepository
	userRepo    user.Repository
}

func NewUserDeletionService(
	accountRepo auth.AccountRepository,
	oauthRepo auth.OAuthConnectionRepository,
	sessionRepo auth.SessionRepository,
	userRepo user.Repository,
) auth.UserDeletionService {
	return &userDeletionServiceImpl{
		accountRepo: accountRepo,
		oauthRepo:   oauthRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

func (s *userDeletionServiceImpl) DeleteUserRecord(
	ctx context.Context,
	exec transaction.Executor,
	userID uuid.UUID,
) error {
	if err := s.userRepo.Delete(ctx, exec, userID); err != nil {
		return fmt.Errorf("failed to soft delete user profile: %w", err)
	}
	if err := s.accountRepo.DeleteByUserID(ctx, exec, userID); err != nil {
		return fmt.Errorf("failed to soft delete account: %w", err)
	}
	if err := s.oauthRepo.DeleteByUserID(ctx, exec, userID); err != nil {
		return fmt.Errorf("failed to soft delete oauth connections: %w", err)
	}
	if err := s.sessionRepo.RevokeAllByUserID(ctx, exec, userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}
