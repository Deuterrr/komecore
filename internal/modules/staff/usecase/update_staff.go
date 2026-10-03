package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	staffDomain "komecore/internal/modules/staff/domain"
	staffRepo "komecore/internal/modules/staff/repository"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type UpdateStaffUsecase struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	staffRepo      staffRepo.StaffRepository
	membershipRepo staffRepo.StaffMembershipRepository
	auditLogger    applogger.AuditLogger
}

func NewUpdateStaffUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	staffRepo staffRepo.StaffRepository,
	membershipRepo staffRepo.StaffMembershipRepository,
	auditLogger applogger.AuditLogger,
) *UpdateStaffUsecase {
	return &UpdateStaffUsecase{
		executor:       executor,
		transactor:     transactor,
		staffRepo:      staffRepo,
		membershipRepo: membershipRepo,
		auditLogger:    auditLogger,
	}
}

type UpdateStaffInput struct {
	ActorAccountID uuid.UUID
	ActorStaffID   uuid.UUID
	StaffID        uuid.UUID
	Name           string
	Description    *string
	LogoUrl        *string
	BannerUrl      *string
}

func (u *UpdateStaffUsecase) Execute(
	ctx context.Context,
	input UpdateStaffInput,
) error {
	if input.Name == "" {
		return apperrors.NewBadRequest("name is required")
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
			Action:   "update_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor membership not found"},
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

	foundAdmin := false
	for _, role := range actorRoles {
		if role.Code == staffDomain.RoleStaffAdmin {
			foundAdmin = true
			break
		}
	}
	if !foundAdmin {
		u.auditLogger.Log(ctx, applogger.AuditEvent{
			Category: "user_action",
			Action:   "update_staff",
			Resource: "staff",
			Outcome:  applogger.OutcomeFailure,
			Metadata: map[string]any{"staff_id": input.StaffID.String(), "reason": "actor lacks admin role"},
		})
		return apperrors.NewForbidden(staffDomain.ErrInsufficientRole.Error())
	}

	staff, err := u.staffRepo.GetByID(ctx, u.executor, input.StaffID)
	if err != nil {
		return fmt.Errorf("failed to retrieve staff: %w", err)
	}
	if staff == nil {
		return apperrors.NewNotFound(staffDomain.ErrNotFoundStaff.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.staffRepo.Update(ctx, exec,
			input.StaffID,
			input.Name,
			input.LogoUrl,
			input.BannerUrl,
		); err != nil {
			return fmt.Errorf("failed to update staff: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	u.auditLogger.Log(ctx, applogger.AuditEvent{
		Category:   "user_action",
		Action:     "update_staff",
		Resource:   "staff",
		ResourceID: input.StaffID.String(),
		Outcome:    applogger.OutcomeSuccess,
		Metadata:   map[string]any{"staff_id": input.StaffID.String(), "name": input.Name},
	})

	return nil
}
