package authsvc

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/authrepo"
	"komecore/internal/modules/user/userrepo"

	"github.com/google/uuid"
)

type userDeletionServiceImpl struct {
	accountRepo authrepo.AccountRepository
	oauthRepo   authrepo.OAuthConnectionRepository
	sessionRepo authrepo.SessionRepository
	userRepo    userrepo.UserRepository
}

func NewUserDeletionService(
	accountRepo authrepo.AccountRepository,
	oauthRepo authrepo.OAuthConnectionRepository,
	sessionRepo authrepo.SessionRepository,
	userRepo userrepo.UserRepository,
) authrepo.UserDeletionService {
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
