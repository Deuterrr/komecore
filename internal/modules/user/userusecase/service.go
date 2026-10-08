package userusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userrepo"
	appclock "komecore/pkg/clock"

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
	GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*userdomain.StaffProfile, error)
}

type ProfileResult struct {
	Customer *userdomain.CustomerProfile
	Staff    *userdomain.StaffProfile
}

type UpdateProfileInput struct {
	Name      *string
	Phone     *string
	AvatarURL *string
}

type UserRepository interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*userdomain.User, error)
	SaveProfile(ctx context.Context, exec transaction.Executor, props userrepo.SaveProfileProps) error
}

type UserService struct {
	executor             transaction.Executor
	transactor           transaction.Transactor
	accountRepo          AccountReader
	staffProfileProvider StaffProfileProvider
	sessionRepo          SessionReader
	userRepo             UserRepository
}

func NewUserService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo AccountReader,
	staffProfileProvider StaffProfileProvider,
	sessionRepo SessionReader,
	userRepo UserRepository,
) *UserService {
	return &UserService{
		executor:             executor,
		transactor:           transactor,
		accountRepo:          accountRepo,
		staffProfileProvider: staffProfileProvider,
		sessionRepo:          sessionRepo,
		userRepo:             userRepo,
	}
}

func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	user, err := s.userRepo.GetByID(ctx, s.executor, id)
	if err != nil {
		return &userdomain.User{}, fmt.Errorf("failed to retrieve user: %w", err)
	}
	return user, nil
}

func (s *UserService) GetCurrentProfile(ctx context.Context, authCtx authctx.AuthContext) (*ProfileResult, error) {
	account, err := s.accountRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperrors.NewNotFound(string(apperrors.ErrTypeNotFound))
	}

	var lastActivityAt *time.Time
	if s.sessionRepo != nil {
		act, err := s.sessionRepo.GetLastActivity(ctx, s.executor, authCtx.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve session: %w", err)
		}
		lastActivityAt = act
	}

	var result ProfileResult
	switch account.Type {
	case authctx.AccountTypeCustomer:
		user, err := s.userRepo.GetByID(ctx, s.executor, authCtx.UserID)
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

		result.Customer = &userdomain.CustomerProfile{
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
		staffProfile, err := s.staffProfileProvider.GetProfileByUserID(
			ctx,
			s.executor,
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

func (s *UserService) UpdateCurrentProfile(
	ctx context.Context,
	authCtx authctx.AuthContext,
	input UpdateProfileInput,
) (*ProfileResult, error) {
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return nil, apperrors.NewBadRequest("name is required")
	}

	account, err := s.accountRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperrors.NewNotFound(string(apperrors.ErrTypeNotFound))
	}

	var result ProfileResult

	err = s.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := s.userRepo.SaveProfile(
				ctx,
				exec,
				userrepo.SaveProfileProps{
					UserID:    authCtx.UserID,
					Name:      input.Name,
					Phone:     input.Phone,
					AvatarURL: input.AvatarURL,
					UpdatedAt: appclock.Now(),
				},
			); err != nil {
				return fmt.Errorf("failed to save user profile: %w", err)
			}

			switch account.Type {
			case authctx.AccountTypeCustomer:
				user, err := s.userRepo.GetByID(ctx, exec, authCtx.UserID)
				if err != nil {
					return fmt.Errorf("failed to retrieve user profile: %w", err)
				}
				if user == nil {
					return apperrors.NewNotFound("user not found")
				}

				customerID := uuid.Nil
				if authCtx.CustomerID != nil {
					customerID = *authCtx.CustomerID
				}

				result.Customer = &userdomain.CustomerProfile{
					ID:        customerID,
					UserID:    user.ID,
					Name:      user.Name,
					Username:  user.Username,
					Phone:     user.Phone,
					AvatarURL: user.AvatarURL,
					CreatedAt: user.CreatedAt,
					UpdatedAt: user.UpdatedAt,
				}

			case authctx.AccountTypeStaff:
				staffProfile, err := s.staffProfileProvider.GetProfileByUserID(ctx, exec, authCtx.UserID)
				if err != nil {
					return fmt.Errorf("failed to retrieve staff profile: %w", err)
					}

				result.Staff = staffProfile
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
