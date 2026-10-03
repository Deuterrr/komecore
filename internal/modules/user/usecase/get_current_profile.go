package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	authenDomain "komecore/internal/modules/auth/domain"
	authenRepo "komecore/internal/modules/auth/repository"
	staffDomain "komecore/internal/modules/staff/domain"
	staffRepo "komecore/internal/modules/staff/repository"
	"komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"

	"github.com/google/uuid"
)

type GetCurrentProfileUsecase struct {
	executor    transaction.Executor
	accountRepo authenRepo.AccountRepository
	userRepo    userRepo.UserRepository
	staffRepo   staffRepo.StaffRepository
	sessionRepo authenRepo.SessionRepository
}

func NewGetCurrentProfileUsecase(
	executor transaction.Executor,
	accountRepo authenRepo.AccountRepository,
	userRepo userRepo.UserRepository,
	staffRepo staffRepo.StaffRepository,
	sessionRepo authenRepo.SessionRepository,
) *GetCurrentProfileUsecase {
	return &GetCurrentProfileUsecase{
		executor:    executor,
		accountRepo: accountRepo,
		userRepo:    userRepo,
		staffRepo:   staffRepo,
		sessionRepo: sessionRepo,
	}
}

type ProfileResult struct {
	Customer *domain.CustomerProfile
	Staff    *staffDomain.StaffProfile
}

func (u *GetCurrentProfileUsecase) Execute(
	ctx context.Context,
	authCtx authenDomain.AuthContext,
) (*ProfileResult, error) {
	account, err := u.accountRepo.GetByUserID(ctx, u.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperrors.NewNotFound(string(apperrors.ErrTypeNotFound))
	}

	session, err := u.sessionRepo.GetByID(ctx, u.executor, authCtx.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	var result ProfileResult
	switch account.Type {
	case authenDomain.AccountTypeCustomer:
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
			LastLoginAt: session.LastActivityAt,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		}
	case authenDomain.AccountTypeStaff:
		staffProfile, err := u.staffRepo.GetProfileByUserID(
			ctx,
			u.executor,
			authCtx.UserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve staff profile: %w", err)
		}

		result.Staff = staffProfile
		result.Staff.LastLoginAt = session.LastActivityAt
	}

	return &result, nil
}
