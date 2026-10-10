package staffusecase

import (
	"context"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveStaffAccount_Success(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	targetAccountUserID := uuid.New()
	staffOwnerUserID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		targetMembership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: targetAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffdomain.Staff{
			ID:     staffID,
			UserID: staffOwnerUserID,
		},
	}

	accountRepo := &mockAccountRepo{
		account: &AccountInfo{
			ID:     targetAccountID,
			UserID: targetAccountUserID,
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
		accountRepo,
		nil,
		nil,
		auditLogger,
	)

	err := uc.RemoveStaffAccount(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, membershipRepo.deleteByAccCalls)
	assert.Equal(t, 1, accountRepo.deleteCalls)
	assert.Equal(t, 1, accountRepo.revokeCalls)
	assert.Equal(t, []uuid.UUID{targetAccountUserID}, accountRepo.revokedUserIDs)
	assert.Len(t, auditLogger.events, 1)
	assert.Equal(t, "remove_staff_account", auditLogger.events[0].Action)
}

func TestRemoveStaffAccount_SelfRemovalBlocked(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	staffID := uuid.New()

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		&mockStaffMembershipRepo{},
		nil,
		nil,
		&mockAccountRepo{},
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.RemoveStaffAccount(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      actorAccountID,
	})

	require.Error(t, err)
	var badReq *apperror.AppError
	require.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)
	assert.Contains(t, badReq.Message, "cannot remove own account")
}

func TestRemoveStaffAccount_NonAdminForbidden(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaff, Name: "Staff"},
		},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		membershipRepo,
		nil,
		nil,
		&mockAccountRepo{},
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.RemoveStaffAccount(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestRemoveStaffAccount_StaffNotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: nil,
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		nil,
		nil,
		&mockAccountRepo{},
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.RemoveStaffAccount(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var notFound *apperror.AppError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, 404, notFound.StatusCode)
}

func TestRemoveStaffAccount_TargetMembershipNotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		targetMembership: nil,
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffdomain.Staff{
			ID:     staffID,
			UserID: uuid.New(),
		},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		nil,
		nil,
		&mockAccountRepo{},
		nil,
		nil,
		&mockAuditLogger{},
	)

	err := uc.RemoveStaffAccount(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var notFound *apperror.AppError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, 404, notFound.StatusCode)
}
