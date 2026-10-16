package staff

import (
	"context"
	"testing"
	"time"

	"komecore/internal/apperror"

	"github.com/google/uuid"
)

func TestListStaffAccounts_Success(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := actorStaffID

	membershipRepo := &mockStaffMembershipRepo{
		membership: &StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []Role{
			{ID: uuid.New(), Code: RoleStaffAdmin, Name: "Staff Admin"},
		},
		accounts: []StaffAccountMember{
			{
				AccountID: actorAccountID,
				UserID:    uuid.New(),
				Email:     "admin@komecore.com",
				Name:      "Admin User",
				Username:  "admin",
				Role: Role{
					ID:   uuid.New(),
					Code: RoleStaffAdmin,
					Name: "Staff Admin",
				},
				CreatedAt: time.Now(),
			},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &Staff{
			ID:     targetStaffID,
			UserID: uuid.New(),
		},
	}

	auditLogger := &mockAuditLogger{}

	uc := NewService(
		&mockExecutor{},
		nil,
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		auditLogger,
	)

	results, err := uc.ListStaffAccounts(ctx, ListStaffAccountsParams{
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
		membership: &StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []Role{
			{ID: uuid.New(), Code: RoleStaff, Name: "Staff"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &Staff{ID: actorStaffID},
	}

	uc := NewService(
		&mockExecutor{},
		nil,
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		&mockAuditLogger{},
	)

	_, err := uc.ListStaffAccounts(ctx, ListStaffAccountsParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        actorStaffID,
	})

	if err == nil {
		t.Fatal("expected error for non-admin actor, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 403 {
		t.Fatalf("expected 403 Forbidden error, got: %v", err)
	}
}

func TestListStaffAccounts_StaffNotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []Role{
			{ID: uuid.New(), Code: RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: nil,
	}

	uc := NewService(
		&mockExecutor{},
		nil,
		staffRepo,
		membershipRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		&mockAuditLogger{},
	)

	_, err := uc.ListStaffAccounts(ctx, ListStaffAccountsParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        uuid.New(),
	})

	if err == nil {
		t.Fatal("expected error for not found staff, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.StatusCode != 404 {
		t.Fatalf("expected 404 Not Found error, got: %v", err)
	}
}
