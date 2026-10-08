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

func TestUpdateCurrentProfileUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	customerID := uuid.New()
	staffID := uuid.New()
	now := time.Now()

	newName := "Bob Updated"
	newPhone := "+6289999999"
	newAvatar := "https://example.com/new.png"

	t.Run("returns bad request when name is empty string or whitespace", func(t *testing.T) {
		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		emptyName := "   "
		input := usecase.UpdateProfileInput{Name: &emptyName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, apperrors.IsBadRequest(err))
	})

	t.Run("returns not found when account does not exist", func(t *testing.T) {
		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: nil}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{Name: &newName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, apperrors.IsNotFound(err))
	})

	t.Run("returns error when account repository fails", func(t *testing.T) {
		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{getByUserIDErr: errors.New("db error")}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{Name: &newName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve account")
	})

	t.Run("success updates customer profile", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeCustomer,
		}
		updatedUser := &userDomain.User{
			ID:        userID,
			Name:      newName,
			Username:  "bob_customer",
			Phone:     &newPhone,
			AvatarURL: &newAvatar,
			CreatedAt: now,
		}

		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: account}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{user: updatedUser}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{
			Name:      &newName,
			Phone:     &newPhone,
			AvatarURL: &newAvatar,
		}
		authCtx := authctx.AuthContext{
			UserID:      userID,
			CustomerID:  &customerID,
			AccountType: authctx.AccountTypeCustomer,
		}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Customer)
		assert.Nil(t, result.Staff)

		assert.Equal(t, customerID, result.Customer.ID)
		assert.Equal(t, userID, result.Customer.UserID)
		assert.Equal(t, newName, result.Customer.Name)
		assert.Equal(t, 1, userRepo.saveProfileCalls)
		assert.Equal(t, 1, userRepo.getByIDCalls)
	})

	t.Run("success updates staff profile", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeStaff,
		}
		updatedStaff := &userDomain.StaffProfile{
			ID:        staffID,
			UserID:    userID,
			Name:      newName,
			Username:  "bob_staff",
			Phone:     &newPhone,
			AvatarURL: &newAvatar,
			CreatedAt: now,
		}

		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: account}
		staffRepo := &mockStaffRepo{profile: updatedStaff}
		userRepo := &mockUserRepo{}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{
			Name:      &newName,
			Phone:     &newPhone,
			AvatarURL: &newAvatar,
		}
		authCtx := authctx.AuthContext{
			UserID:      userID,
			StaffID:     &staffID,
			AccountType: authctx.AccountTypeStaff,
		}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Staff)
		assert.Nil(t, result.Customer)

		assert.Equal(t, staffID, result.Staff.ID)
		assert.Equal(t, newName, result.Staff.Name)
		assert.Equal(t, 1, userRepo.saveProfileCalls)
		assert.Equal(t, 1, staffRepo.getProfileCalls)
	})

	t.Run("returns error when SaveProfile fails", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeCustomer,
		}

		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: account}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{saveProfileError: errors.New("save error")}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{Name: &newName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save user profile")
	})

	t.Run("returns not found error when customer user is missing after save", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeCustomer,
		}

		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: account}
		staffRepo := &mockStaffRepo{}
		userRepo := &mockUserRepo{user: nil}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{Name: &newName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, apperrors.IsNotFound(err))
	})

	t.Run("returns error when staff profile retrieval fails", func(t *testing.T) {
		account := &usecase.UserAccount{
			Type: authctx.AccountTypeStaff,
		}

		exec := &mockExecutor{}
		tx := &mockTransactor{}
		accountRepo := &mockAccountRepo{account: account}
		staffRepo := &mockStaffRepo{getProfileErr: errors.New("staff get error")}
		userRepo := &mockUserRepo{}

		svc := usecase.NewUserService(exec, tx, accountRepo, staffRepo, nil, userRepo)
		input := usecase.UpdateProfileInput{Name: &newName}
		authCtx := authctx.AuthContext{UserID: userID}

		result, err := svc.UpdateCurrentProfile(ctx, authCtx, input)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve staff profile")
	})
}
