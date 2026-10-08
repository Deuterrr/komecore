package userusecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/modules/user/userdomain"
	"komecore/internal/modules/user/userusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserUsecase_ByID(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	phone := "+628123456789"
	avatar := "https://example.com/avatar.jpg"
	now := time.Now()

	expectedUser := &userdomain.User{
		ID:        userID,
		Name:      "Alice Smith",
		Username:  "alicesmith",
		Phone:     &phone,
		Role:      userdomain.RoleCustomer,
		AvatarURL: &avatar,
		CreatedAt: now,
	}

	t.Run("success returns user", func(t *testing.T) {
		repo := &mockUserRepo{
			user: expectedUser,
		}
		exec := &mockExecutor{}

		svc := userusecase.NewUserService(exec, nil, nil, nil, nil, repo)
		result, err := svc.GetUserByID(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, expectedUser, result)
		assert.Equal(t, 1, repo.getByIDCalls)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := &mockUserRepo{
			getByIDError: errors.New("user not found in db"),
		}
		exec := &mockExecutor{}

		svc := userusecase.NewUserService(exec, nil, nil, nil, nil, repo)
		result, err := svc.GetUserByID(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve user")
		assert.Equal(t, &userdomain.User{}, result)
		assert.Equal(t, 1, repo.getByIDCalls)
	})
}
