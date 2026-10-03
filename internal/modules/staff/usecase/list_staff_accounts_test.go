package usecase

import (
	"context"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	staffDomain "komecore/internal/modules/staff/domain"

	"github.com/google/uuid"
)

func TestListStaffAccounts_Success(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := actorStaffID

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
		accounts: []staffDomain.StaffAccountMember{
			{
				AccountID: actorAccountID,
				UserID:    uuid.New(),
				Email:     "admin@komecore.com",
				Name:      "Admin User",
				Username:  "admin",
				Role: staffDomain.Role{
					ID:   uuid.New(),
					Code: staffDomain.RoleStaffAdmin,
					Name: "Staff Admin",
				},
				CreatedAt: time.Now(),
			},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffDomain.Staff{
			ID:     targetStaffID,
			UserID: uuid.New(),
		},
	}

	auditLogger := &mockAuditLogger{}

	uc := NewListStaffAccountsUsecase(
		&mockExecutor{},
		staffRepo,
		membershipRepo,
		auditLogger,
	)

	results, err := uc.Execute(ctx, ListStaffAccountsParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 account, got: %d", len(results))
	}
	if results[0].Email != "admin@komecore.com" {
		t.Errorf("expected email admin@komecore.com, got: %s", results[0].Email)
	}
}

func TestListStaffAccounts_UnauthorizedRole(t *testing.T) {
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
			{ID: uuid.New(), Code: staffDomain.RoleStaff, Name: "Staff"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffDomain.Staff{ID: actorStaffID},
	}

	uc := NewListStaffAccountsUsecase(
		&mockExecutor{},
		staffRepo,
		membershipRepo,
		&mockAuditLogger{},
	)

	_, err := uc.Execute(ctx, ListStaffAccountsParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        actorStaffID,
	})

	if err == nil {
		t.Fatal("expected error for non-admin actor, got nil")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.StatusCode != 403 {
		t.Fatalf("expected 403 Forbidden error, got: %v", err)
	}
}

func TestListStaffAccounts_StaffNotFound(t *testing.T) {
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

	uc := NewListStaffAccountsUsecase(
		&mockExecutor{},
		staffRepo,
		membershipRepo,
		&mockAuditLogger{},
	)

	_, err := uc.Execute(ctx, ListStaffAccountsParams{
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
