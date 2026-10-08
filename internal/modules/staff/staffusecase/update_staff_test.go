package staffusecase

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
)

func TestUpdateStaff_Success(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := actorStaffID

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffdomain.Staff{
			ID:     targetStaffID,
			UserID: uuid.New(),
		},
	}

	auditLogger := &mockAuditLogger{}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		auditLogger,
	)

	newName := "Updated Branch"
	err := uc.UpdateStaff(ctx, UpdateStaffInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
		Name:           newName,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if staffRepo.updateCalls != 1 {
		t.Errorf("expected 1 update call, got: %d", staffRepo.updateCalls)
	}
	if len(auditLogger.events) != 1 {
		t.Errorf("expected 1 audit event, got: %d", len(auditLogger.events))
	}
}

func TestUpdateStaff_EmptyName(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		&mockStaffMembershipRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.UpdateStaff(ctx, UpdateStaffInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        actorStaffID,
		Name:           "",
	})

	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.StatusCode != 400 {
		t.Fatalf("expected 400 Bad Request error, got: %v", err)
	}
}

func TestUpdateStaff_UnauthorizedRole(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaff, Name: "Staff"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffdomain.Staff{ID: actorStaffID},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.UpdateStaff(ctx, UpdateStaffInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        actorStaffID,
		Name:           "Updated Name",
	})

	if err == nil {
		t.Fatal("expected error for non-admin, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.StatusCode != 403 {
		t.Fatalf("expected 403 Forbidden error, got: %v", err)
	}
}
