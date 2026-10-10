package staffusecase

import (
	"context"
	"testing"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customMockAccountRepo struct {
	mockAccountRepo
	existingAccount       *AccountInfo
	existingAccountByUser *AccountInfo
	createCalls           int
}

func (m *customMockAccountRepo) GetByEmail(ctx context.Context, exec transaction.Executor, email string) (*AccountInfo, error) {
	return m.existingAccount, nil
}

func (m *customMockAccountRepo) GetByUserID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*AccountInfo, error) {
	return m.existingAccountByUser, nil
}

func (m *customMockAccountRepo) CreateStaffAccount(ctx context.Context, exec transaction.Executor, input CreateAccountInput) error {
	m.createCalls++
	return nil
}

func TestAddStaffAccountUsecase_Success(t *testing.T) {
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := uuid.New()
	targetUserID := uuid.New()
	roleID := uuid.New()

	memRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: roleID, Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}
	staffR := &mockStaffRepo{
		staff: &staffdomain.Staff{
			ID:        targetStaffID,
			UserID:    targetUserID,
			CreatedAt: time.Now(),
		},
	}
	accRepo := &customMockAccountRepo{}
	roleR := &mockRoleRepo{
		role: &staffdomain.Role{ID: roleID, Code: staffdomain.RoleStaff, Name: "Staff"},
	}
	pwHash := &mockPwHasher{}
	exec := &mockExecutor{}
	tx := &mockTransactor{}
	audit := &mockAuditLogger{}

	uc := NewStaffService(exec, tx, staffR, memRepo, roleR, nil, accRepo, pwHash, nil, audit)

	params := AddStaffAccountParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
		Email:          "newstaff@komecore.com",
		Password:       "password123",
	}

	err := uc.AddStaffAccount(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, 1, accRepo.createCalls)
	assert.Equal(t, 1, memRepo.saveCalls)
	assert.Len(t, audit.events, 1)
	assert.Equal(t, "add_staff_account", audit.events[0].Action)
}

func TestAddStaffAccountUsecase_ValidationErrors(t *testing.T) {
	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		&mockStaffMembershipRepo{},
		&mockRoleRepo{},
		nil,
		&customMockAccountRepo{},
		&mockPwHasher{},
		nil,
		&mockAuditLogger{},
	)

	// Missing email
	err := uc.AddStaffAccount(context.Background(), AddStaffAccountParams{
		Email:    "",
		Password: "password123",
	})
	assert.Error(t, err)
	var badReq *apperror.AppError
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)

	// Missing password
	err = uc.AddStaffAccount(context.Background(), AddStaffAccountParams{
		Email:    "staff@komecore.com",
		Password: "",
	})
	assert.Error(t, err)
	assert.ErrorAs(t, err, &badReq)
	assert.Equal(t, 400, badReq.StatusCode)
}

func TestAddStaffAccountUsecase_NonAdminForbidden(t *testing.T) {
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := uuid.New()

	memRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaff, Name: "Staff"},
		},
	}
	staffR := &mockStaffRepo{
		staff: &staffdomain.Staff{ID: targetStaffID, UserID: uuid.New()},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffR,
		memRepo,
		&mockRoleRepo{},
		nil,
		&customMockAccountRepo{},
		&mockPwHasher{},
		nil,
		&mockAuditLogger{},
	)

	err := uc.AddStaffAccount(context.Background(), AddStaffAccountParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
		Email:          "staff@komecore.com",
		Password:       "password123",
	})
	assert.Error(t, err)
	var appErr *apperror.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestAddStaffAccountUsecase_DuplicateEmail(t *testing.T) {
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := uuid.New()

	memRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}
	accRepo := &customMockAccountRepo{
		existingAccount: &AccountInfo{
			ID:    uuid.New(),
			Email: "taken@komecore.com",
		},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		&mockStaffRepo{},
		memRepo,
		&mockRoleRepo{},
		nil,
		accRepo,
		&mockPwHasher{},
		nil,
		&mockAuditLogger{},
	)

	err := uc.AddStaffAccount(context.Background(), AddStaffAccountParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
		Email:          "taken@komecore.com",
		Password:       "password123",
	})
	assert.Error(t, err)
	var appErr *apperror.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
}

func TestAddStaffAccountUsecase_AlreadyBoundConflict(t *testing.T) {
	actorAccountID := uuid.New()
	actorStaffID := uuid.New()
	targetStaffID := uuid.New()
	targetUserID := uuid.New()

	memRepo := &mockStaffMembershipRepo{
		membership: &staffdomain.StaffMembership{
			ID:        uuid.New(),
			StaffID:   actorStaffID,
			AccountID: actorAccountID,
		},
		roles: []staffdomain.Role{
			{ID: uuid.New(), Code: staffdomain.RoleStaffAdmin, Name: "Staff Admin"},
		},
	}
	staffR := &mockStaffRepo{
		staff: &staffdomain.Staff{ID: targetStaffID, UserID: targetUserID},
	}
	accRepo := &customMockAccountRepo{
		existingAccountByUser: &AccountInfo{
			ID:     uuid.New(),
			UserID: targetUserID,
			Email:  "alreadybound@komecore.com",
		},
	}

	uc := NewStaffService(
		&mockExecutor{},
		&mockTransactor{},
		staffR,
		memRepo,
		&mockRoleRepo{},
		nil,
		accRepo,
		&mockPwHasher{},
		nil,
		&mockAuditLogger{},
	)

	err := uc.AddStaffAccount(context.Background(), AddStaffAccountParams{
		ActorAccountID: actorAccountID,
		ActorStaffID:   actorStaffID,
		StaffID:        targetStaffID,
		Email:          "another@komecore.com",
		Password:       "password123",
	})
	assert.Error(t, err)
	var appErr *apperror.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 409, appErr.StatusCode)
	assert.Contains(t, appErr.Message, "1 account per user limit")
}
