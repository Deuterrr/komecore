package usecase

import (
	"context"
	"fmt"
	"strings"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	authenDomain "komecore/internal/modules/auth/domain"
	authenRepo "komecore/internal/modules/auth/repository"
	staffRepo "komecore/internal/modules/staff/repository"
	"komecore/internal/modules/user/domain"
	userRepo "komecore/internal/modules/user/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type UpdateCurrentProfileUsecase struct {
	executor    transaction.Executor
	transactor  transaction.Transactor
	accountRepo authenRepo.AccountRepository
	staffRepo   staffRepo.StaffRepository
	userRepo    userRepo.UserRepository
}

func NewUpdateCurrentProfileUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo authenRepo.AccountRepository,
	staffRepo staffRepo.StaffRepository,
	userRepo userRepo.UserRepository,
) *UpdateCurrentProfileUsecase {
	return &UpdateCurrentProfileUsecase{
		executor:    executor,
		transactor:  transactor,
		accountRepo: accountRepo,
		staffRepo:   staffRepo,
		userRepo:    userRepo,
	}
}

type UpdateProfileInput struct {
	Name      *string
	Phone     *string
	AvatarURL *string
}

func (u *UpdateCurrentProfileUsecase) Execute(
	ctx context.Context,
	authCtx authenDomain.AuthContext,
	input UpdateProfileInput,
) (*ProfileResult, error) {
	if input.Name != nil &&
		strings.TrimSpace(*input.Name) == "" {
		return nil, apperrors.NewBadRequest("name is required")
	}

	account, err := u.accountRepo.GetByUserID(ctx, u.executor, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	if account == nil {
		return nil, apperrors.NewNotFound(string(apperrors.ErrTypeNotFound))
	}

	var result ProfileResult

	err = u.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := u.userRepo.SaveProfile(
				ctx,
				exec,
				userRepo.SaveProfileProps{
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
			case authenDomain.AccountTypeCustomer:
				user, err := u.userRepo.GetByID(ctx, exec, authCtx.UserID)
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

				result.Customer = &domain.CustomerProfile{
					ID:        customerID,
					UserID:    user.ID,
					Name:      user.Name,
					Username:  user.Username,
					Phone:     user.Phone,
					AvatarURL: user.AvatarURL,
					CreatedAt: user.CreatedAt,
					UpdatedAt: user.UpdatedAt,
				}

			case authenDomain.AccountTypeStaff:
				staffProfile, err := u.staffRepo.GetProfileByUserID(ctx, exec, authCtx.UserID)
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
