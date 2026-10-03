package service

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	authenRepo "komecore/internal/modules/auth/repository"
	userRepo "komecore/internal/modules/user/repository"

	"github.com/google/uuid"
)

type userDeletionServiceImpl struct {
	accountRepo authenRepo.AccountRepository
	oauthRepo   authenRepo.OAuthConnectionRepository
	sessionRepo authenRepo.SessionRepository
	userRepo    userRepo.UserRepository
}

func NewUserDeletionService(
	accountRepo authenRepo.AccountRepository,
	oauthRepo authenRepo.OAuthConnectionRepository,
	sessionRepo authenRepo.SessionRepository,
	userRepo userRepo.UserRepository,
) authenRepo.UserDeletionService {
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
