package usecase

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	authenDomain "komecore/internal/modules/auth/domain"
	staffDomain "komecore/internal/modules/staff/domain"

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
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		targetMembership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: targetAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffDomain.Staff{
			ID:     staffID,
			UserID: staffOwnerUserID,
		},
	}

	accountRepo := &mockAccountRepo{
		account: &authenDomain.Account{
			ID:     targetAccountID,
			UserID: targetAccountUserID,
		},
	}

	sessionRepo := &mockSessionRepo{}
	auditLogger := &mockAuditLogger{}

	uc := NewRemoveStaffAccountUsecase(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		accountRepo,
		sessionRepo,
		auditLogger,
	)

	err := uc.Execute(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, membershipRepo.deleteByAccCalls)
	assert.Equal(t, 1, accountRepo.deleteCalls)
	assert.Equal(t, 1, sessionRepo.revokeCalls)
	assert.Equal(t, []uuid.UUID{targetAccountUserID}, sessionRepo.revokedUserIDs)
	assert.Len(t, auditLogger.events, 1)
	assert.Equal(t, "remove_staff_account", auditLogger.events[0].Action)
}

func TestRemoveStaffAccount_SelfRemovalBlocked(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	staffID := uuid.New()

	uc := NewRemoveStaffAccountUsecase(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		&mockStaffMembershipRepo{},
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockAuditLogger{},
	)

	err := uc.Execute(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      actorAccountID,
	})

	require.Error(t, err)
	var badReq *apperrors.AppError
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
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaff, Name: "Staff"},
		},
	}

	uc := NewRemoveStaffAccountUsecase(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		membershipRepo,
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockAuditLogger{},
	)

	err := uc.Execute(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestRemoveStaffAccount_StaffNotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: nil,
	}

	uc := NewRemoveStaffAccountUsecase(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockAuditLogger{},
	)

	err := uc.Execute(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var notFound *apperrors.AppError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, 404, notFound.StatusCode)
}

func TestRemoveStaffAccount_TargetMembershipNotFound(t *testing.T) {
	ctx := context.Background()
	actorAccountID := uuid.New()
	targetAccountID := uuid.New()
	staffID := uuid.New()

	membershipRepo := &mockStaffMembershipRepo{
		membership: &staffDomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   staffID,
			AccountID: actorAccountID,
		},
		targetMembership: nil,
		roles: []staffDomain.Role{
			{ID: uuid.New(), Code: staffDomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}

	staffRepo := &mockStaffRepo{
		staff: &staffDomain.Staff{
			ID:     staffID,
			UserID: uuid.New(),
		},
	}

	uc := NewRemoveStaffAccountUsecase(
		&mockExecutor{},
		&mockTransactor{},
		staffRepo,
		membershipRepo,
		&mockAccountRepo{},
		&mockSessionRepo{},
		&mockAuditLogger{},
	)

	err := uc.Execute(ctx, RemoveStaffAccountInput{
		ActorAccountID: actorAccountID,
		ActorStaffID:   staffID,
		StaffID:        staffID,
		AccountID:      targetAccountID,
	})

	require.Error(t, err)
	var notFound *apperrors.AppError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, 404, notFound.StatusCode)
}
