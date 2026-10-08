package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	staffDomain "komecore/internal/modules/staff/domain"
	"komecore/internal/modules/staff/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type AddStaffAccountUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	accountRepo    AccountManager
	pwHasher       PasswordHasher
	staffRepo      repository.StaffRepository
	membershipRepo repository.StaffMembershipRepository
	roleRepo       repository.RoleRepository
	auditLogger    applogger.AuditLogger
}

func NewAddStaffAccountUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	accountRepo AccountManager,
	pwHasher PasswordHasher,
	staffRepo repository.StaffRepository,
	membershipRepo repository.StaffMembershipRepository,
	roleRepo repository.RoleRepository,
	auditLogger applogger.AuditLogger,
) *AddStaffAccountUsecase {
	return &AddStaffAccountUsecase{
		executor:       executor,
		transactor:     transactor,
		accountRepo:    accountRepo,
		pwHasher:       pwHasher,
		staffRepo:      staffRepo,
		membershipRepo: membershipRepo,
		roleRepo:       roleRepo,
		auditLogger:    auditLogger,
	}
}

type AddStaffAccountParams struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
	Email          string
	Password       string
}

func (u *AddStaffAccountUsecase) Execute(
	ctx context.Context,
	input AddStaffAccountParams,
) error {
	if input.Email == "" {
		return apperrors.NewBadRequest("email is required")
	}
	if input.Password == "" {
		return apperrors.NewBadRequest("password is required")
	}

	actorMembership, err := u.membershipRepo.GetByAccountIDAndStaffID(ctx, u.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to verify actor membership: %w", err)
	}
	if actorMembership == nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "actor membership not found"},
		})
		return apperrors.NewForbidden(staffDomain.ErrInsufficientRole.Error())
	}

	actorRoles, err := u.membershipRepo.ListRolesByAccountIDAndStaffID(ctx, u.executor,
		input.ActorAccountID,
		input.ActorStaffID,
	)
	if err != nil {
		return fmt.Errorf("failed to retrieve actor roles: %w", err)
	}

	found := false
	for _, role := range actorRoles {
		if role.Code == staffDomain.RoleStaffAdmin {
			found = true
			break
		}
	}
	if !found {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "actor lacks admin role"},
		})
		return apperrors.NewForbidden(staffDomain.ErrInsufficientRole.Error())
	}

	existingAcc, err := u.accountRepo.GetByEmail(ctx, u.executor, input.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing account: %w", err)
	}
	if existingAcc != nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "email already exists"},
		})
		return apperrors.NewConflict("an account with this email already exists")
	}

	existingStaff, err := u.staffRepo.GetByID(ctx, u.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to check existing staff: %w", err)
	}
	if existingStaff == nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "staff not found"},
		})
		return apperrors.NewNotFound(staffDomain.ErrNotFoundStaff.Error())
	}

	existingUserAcc, err := u.accountRepo.GetByUserID(ctx, u.executor, existingStaff.UserID)
	if err != nil {
		return fmt.Errorf("failed to check existing staff account: %w", err)
	}
	if existingUserAcc != nil {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "add_staff_account",
			Resource: "staff_account",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "email": input.Email, "reason": "staff user already has a bound account"},
		})
		return apperrors.NewConflict("this staff entity already has a bound account (1 account per user limit)")
	}

	staffRole, err := u.roleRepo.GetByCode(ctx, u.executor, staffDomain.RoleStaff)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff role: %w", err)
	}
	if staffRole == nil {
		return fmt.Errorf("staff role not found in database")
	}

	now := appclock.Now()
	newAccountID := uuid.New()

	hash, err := u.pwHasher.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("failed to generate placeholder password: %w", err)
	}

	newAccountInput := CreateAccountInput{
		ID:        newAccountID,
		UserID:    existingStaff.UserID,
		Email:     input.Email,
		Password:  hash,
		CreatedAt: now,
	}

	newMembership := staffDomain.StaffMembership{
		ID:        uuid.New(),
		StaffID:   existingStaff.ID,
		AccountID: newAccountID,
		RoleID:    staffRole.ID,
		CreatedBy: input.ActorAccountID,
		CreatedAt: now,
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.accountRepo.CreateStaffAccount(ctx, exec, newAccountInput); err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
		if err := u.membershipRepo.Save(ctx, exec, newMembership); err != nil {
			return fmt.Errorf("failed to save membership: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "add_staff_account",
		Resource:   "staff_account",
		ResourceID: newAccountID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "email": input.Email},
	})

	return nil
}
