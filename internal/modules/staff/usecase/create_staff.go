package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/domain"
	"komecore/internal/modules/staff/repository"
	userRepo "komecore/internal/modules/user/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type CreateStaffUsecase struct {
	staffRepo   repository.StaffRepository
	userRepo    userRepo.UserRepository
	executor    transaction.Executor
	transactor  transaction.Transactor
	auditLogger applogger.AuditLogger
}

func NewCreateStaffUsecase(
	staffRepo repository.StaffRepository,
	userRepo userRepo.UserRepository,
	executor transaction.Executor,
	transactor transaction.Transactor,
	auditLogger applogger.AuditLogger,
) *CreateStaffUsecase {
	return &CreateStaffUsecase{
		staffRepo:   staffRepo,
		userRepo:    userRepo,
		executor:    executor,
		transactor:  transactor,
		auditLogger: auditLogger,
	}
}

type CreateStaffInput struct {
	Name        string
	Username    string
	Description *string
	LogoUrl     *string
	BannerUrl   *string
}

func (u *CreateStaffUsecase) Execute(
	ctx context.Context,
	input CreateStaffInput,
) error {
	if input.Name == "" {
		return apperrors.NewBadRequest("name is required")
	}
	if input.Username == "" {
		return apperrors.NewBadRequest("username is required")
	}

	existingUser, err := u.userRepo.GetByUsername(ctx, u.executor, input.Username)
	if err != nil {
		return fmt.Errorf("failed to check existing username: %w", err)
	}
	if existingUser != nil {
		return apperrors.NewConflict("a user with this username already exists")
	}

	now := appclock.Now()
	newUserID := uuid.New()
	newStaffID := uuid.New()

	newUser := userRepo.CreateUserProps{
		ID:        newUserID,
		Name:      input.Name,
		Username:  input.Username,
		Phone:     nil,
		CreatedAt: now,
	}
	if input.LogoUrl != nil {
		newUser.AvatarURL = input.LogoUrl
	}

	staff := domain.Staff{
		ID:        newStaffID,
		UserID:    newUserID,
		CreatedAt: now,
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.userRepo.CreateUser(ctx, exec, newUser); err != nil {
			return fmt.Errorf("failed to create user for staff: %w", err)
		}
		if err := u.staffRepo.Create(ctx, exec, staff); err != nil {
			return fmt.Errorf("failed to create staff entity: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "create_staff_profile",
		Resource:   "staff",
		ResourceID: staff.ID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"name": input.Name, "username": input.Username, "user_id": newUserID.String()},
	})

	return nil
}
