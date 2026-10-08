package usecase

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	staffDomain "komecore/internal/modules/staff/domain"

	"github.com/google/uuid"
)

func TestDeleteStaff_Success(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := actorStaffID
	staffUserID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffDomain.Staff{
			ID:     targetStaffID,
			UserID: staffUserID,
		},
	}

	userDeletionService := &mockUserDeletionService{}
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
		userDeletionService,
		auditLogger,
	)

	err := uc.DeleteStaff(ctx, DeleteStaffInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if staffRepo.deleteCalls != 1 {
		t.Errorf("expected 1 staff delete call, got: %d", staffRepo.deleteCalls)
	}
	if membershipRepo.deleteByStaffCalls != 1 {
		t.Errorf("expected 1 membership delete call, got: %d", membershipRepo.deleteByStaffCalls)
	}
	if len(userDeletionService.deletedUsers) != 1 || userDeletionService.deletedUsers[0] != staffUserID {
		t.Errorf("expected user deletion for %s, got: %v", staffUserID, userDeletionService.deletedUsers)
	}
	if len(auditLogger.events) != 1 {
		t.Errorf("expected 1 audit event, got: %d", len(auditLogger.events))
	}
}

func TestDeleteStaff_NotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: nil,
	}

	userDeletionService := &mockUserDeletionService{}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		userDeletionService,
		&mockAuditLogger{},
	)

	err := uc.DeleteStaff(ctx, DeleteStaffInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        uuid.New(),
	})

	if err == nil {
		t.Fatal("expected error for not found staff, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.StatusCode != 404 {
		t.Fatalf("expected 404 Not Found error, got: %v", err)
	}
}
