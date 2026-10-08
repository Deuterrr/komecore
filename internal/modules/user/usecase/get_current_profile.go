package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/common/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"

	"github.com/google/uuid"
)

type UserAccount struct {
	Type authctx.AccountType
}

type AccountReader interface {
	GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*UserAccount, error)
}

type SessionReader interface {
	GetLastActivity(ctx context.Context, exec transaction.Executor, sessionID uuid.UUID) (*time.Time, error)
}

type StaffProfileProvider interface {
	GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*domain.StaffProfile, error)
}

type GetCurrentProfileUsecase struct {
	executor             transaction.Executor
	accountRepo          AccountReader
	userRepo             userRepo.UserRepository
	staffProfileProvider StaffProfileProvider
	sessionRepo          SessionReader
}

func NewGetCurrentProfileUsecase(
	executor transaction.Executor,
	accountRepo AccountReader,
	userRepo userRepo.UserRepository,
	staffProfileProvider StaffProfileProvider,
	sessionRepo SessionReader,
) *GetCurrentProfileUsecase {
	return &GetCurrentProfileUsecase{
		executor:             executor,
		accountRepo:          accountRepo,
		userRepo:             userRepo,
		staffProfileProvider: staffProfileProvider,
		sessionRepo:          sessionRepo,
	}
}

type ProfileResult struct {
	Customer *domain.CustomerProfile
	Staff    *domain.StaffProfile
}

func (u *GetCurrentProfileUsecase) Execute(
	ctx context.Context,
	authCtx authctx.AuthContext,
) (*ProfileResult, error) {
	account, err := u.accountRepo.GetByUserID(ctx, u.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperrors.NewNotFound(string(apperrors.ErrTypeNotFound))
	}

	lastActivityAt, err := u.sessionRepo.GetLastActivity(ctx, u.executor, authCtx.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	var result ProfileResult
	switch account.Type {
	case authctx.AccountTypeCustomer:
		user, err := u.userRepo.GetByID(ctx, u.executor, authCtx.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
		}
		if user == nil {
			return nil, apperrors.NewNotFound("user not found")
		}

		customerID := uuid.Nil
		if authCtx.CustomerID != nil {
			customerID = *authCtx.CustomerID
		}

		result.Customer = &domain.CustomerProfile{
			ID:          customerID,
			UserID:      user.ID,
			Name:        user.Name,
			Username:    user.Username,
			Phone:       user.Phone,
			AvatarURL:   user.AvatarURL,
			LastLoginAt: lastActivityAt,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		}
	case authctx.AccountTypeStaff:
		staffProfile, err := u.staffProfileProvider.GetProfileByUserID(
			ctx,
			u.executor,
			authCtx.UserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve staff profile: %w", err)
		}

		result.Staff = staffProfile
		if result.Staff != nil {
			result.Staff.LastLoginAt = lastActivityAt
		}
	}

	return &result, nil
}
