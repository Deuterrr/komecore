package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	userDomain "komecore/internal/modules/user/domain"
	"komecore/internal/modules/user/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentProfileUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	sessionID := uuid.New()
	customerID := uuid.New()
	staffID := uuid.New()
	now := time.Now()
	phone := "+62812345678"
	avatar := "https://example.com/avatar.png"

	t.Run("success returns customer profile", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeCustomer,
		}
		user := &userDomain.User{
			ID:        userID,
			Name:      "Customer John",
			Username:  "john_customer",
			Phone:     &phone,
			AvatarURL: &avatar,
			CreatedAt: now,
		}

		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{user: user}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{
			UserID:      userID,
			SessionID:   sessionID,
			CustomerID:  &customerID,
			AccountType: authctx.AccountTypeCustomer,
		}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Customer)
		assert.Nil(t, result.Staff)

		assert.Equal(t, customerID, result.Customer.ID)
		assert.Equal(t, userID, result.Customer.UserID)
		assert.Equal(t, "Customer John", result.Customer.Name)
		assert.Equal(t, &now, result.Customer.LastLoginAt)
	})

	t.Run("success returns staff profile", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeStaff,
		}
		staffProfile := &userDomain.StaffProfile{
			ID:        staffID,
			UserID:    userID,
			Name:      "Staff Sarah",
			Username:  "sarah_staff",
			Phone:     &phone,
			AvatarURL: &avatar,
			CreatedAt: now,
		}

		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{}
		staffRepo := &mockStaffRepo{profile: staffProfile}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{
			UserID:      userID,
			SessionID:   sessionID,
			StaffID:     &staffID,
			AccountType: authctx.AccountTypeStaff,
		}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Staff)
		assert.Nil(t, result.Customer)

		assert.Equal(t, staffID, result.Staff.ID)
		assert.Equal(t, "Staff Sarah", result.Staff.Name)
		assert.Equal(t, &now, result.Staff.LastLoginAt)
	})

	t.Run("returns not found when account does not exist", func(t *testing.T) {
		accountRepo := &mockAccountRepo{account: nil}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, apperrors.IsNotFound(err))
	})

	t.Run("returns error when account repository fails", func(t *testing.T) {
		accountRepo := &mockAccountRepo{getByUserIDErr: errors.New("db error")}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve account")
	})

	t.Run("returns error when session repository fails", func(t *testing.T) {
		account := &usecase.UserAccount{Type: authctx.AccountTypeCustomer}
		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{getByIDErr: errors.New("session db error")}
		userRepo := &mockUserRepo{}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve session")
	})

	t.Run("returns not found when customer user does not exist", func(t *testing.T) {
		account := &usecase.UserAccount{Type: authctx.AccountTypeCustomer}
		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{user: nil}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, apperrors.IsNotFound(err))
	})

	t.Run("returns error when customer user repo fails", func(t *testing.T) {
		account := &usecase.UserAccount{Type: authctx.AccountTypeCustomer}
		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{getByIDError: errors.New("user db error")}
		staffRepo := &mockStaffRepo{}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve user profile")
	})

	t.Run("returns error when staff repo fails", func(t *testing.T) {
		account := &usecase.UserAccount{Type: authctx.AccountTypeStaff}
		accountRepo := &mockAccountRepo{account: account}
		sessionRepo := &mockSessionRepo{lastActivity: &now}
		userRepo := &mockUserRepo{}
		staffRepo := &mockStaffRepo{getProfileErr: errors.New("staff db error")}
		exec := &mockExecutor{}

		svc := usecase.NewUserService(exec, nil, accountRepo, staffRepo, sessionRepo, userRepo)
		authCtx := authctx.AuthContext{UserID: userID, SessionID: sessionID}

		result, err := svc.GetCurrentProfile(ctx, authCtx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve staff profile")
	})
}
