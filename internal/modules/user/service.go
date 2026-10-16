package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/authctx"
	transaction "komecore/internal/infra/transactor"
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
	GetProfileByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*StaffProfile, error)
}

type Service struct {
	executor             transaction.Executor
	transactor           transaction.Transactor
	accountRepo          AccountReader
	staffProfileProvider StaffProfileProvider
	sessionRepo          SessionReader
	userRepo             Repository
}

func NewService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo AccountReader,
	staffProfileProvider StaffProfileProvider,
	sessionRepo SessionReader,
	userRepo Repository,
) *Service {
	return &Service{
		executor:             executor,
		transactor:           transactor,
		accountRepo:          accountRepo,
		staffProfileProvider: staffProfileProvider,
		sessionRepo:          sessionRepo,
		userRepo:             userRepo,
	}
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := s.userRepo.GetByID(ctx, s.executor, id)
	if err != nil {
		return &User{}, fmt.Errorf("failed to retrieve user: %w", err)
	}
	return u, nil
}

func (s *Service) GetCurrentProfile(ctx context.Context, authCtx authctx.AuthContext) (*ProfileResult, error) {
	account, err := s.accountRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperror.NewNotFound(string(apperror.ErrTypeNotFound))
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
		u, err := s.userRepo.GetByID(ctx, s.executor, authCtx.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
		}
		if u == nil {
			return nil, apperror.NewNotFound("user not found")
		}

		customerID := uuid.Nil
		if authCtx.CustomerID != nil {
			customerID = *authCtx.CustomerID
		}

		result.Customer = &CustomerProfile{
			ID:          customerID,
			UserID:      u.ID,
			Name:        u.Name,
			Username:    u.Username,
			Phone:       u.Phone,
			AvatarURL:   u.AvatarURL,
			LastLoginAt: lastActivityAt,
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
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

type ProfileResult struct {
	Customer *CustomerProfile
	Staff    *StaffProfile
}

type UpdateProfileInput struct {
	Name      *string
	Phone     *string
	AvatarURL *string
}

func (s *Service) UpdateCurrentProfile(ctx context.Context, authCtx authctx.AuthContext, input UpdateProfileInput) (*ProfileResult, error) {
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return nil, apperror.NewBadRequest("name is required")
	}

	account, err := s.accountRepo.GetByUserID(ctx, s.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}
	if account == nil {
		return nil, apperror.NewNotFound(string(apperror.ErrTypeNotFound))
	}

	var result ProfileResult

	err = s.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := s.userRepo.SaveProfile(
				ctx,
				exec,
				SaveProfileProps{
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
				u, err := s.userRepo.GetByID(ctx, exec, authCtx.UserID)
				if err != nil {
					return fmt.Errorf("failed to retrieve user profile: %w", err)
				}
				if u == nil {
					return apperror.NewNotFound("user not found")
				}

				customerID := uuid.Nil
				if authCtx.CustomerID != nil {
					customerID = *authCtx.CustomerID
				}

				result.Customer = &CustomerProfile{
					ID:        customerID,
					UserID:    u.ID,
					Name:      u.Name,
					Username:  u.Username,
					Phone:     u.Phone,
					AvatarURL: u.AvatarURL,
					CreatedAt: u.CreatedAt,
					UpdatedAt: u.UpdatedAt,
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
