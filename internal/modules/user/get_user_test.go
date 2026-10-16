package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/modules/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUser_ByID(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	phone := "+628123456789"
	avatar := "https://example.com/avatar.jpg"
	now := time.Now()

	expectedUser := &user.User{
		ID:        userID,
		Name:      "Alice Smith",
		Username:  "alicesmith",
		Phone:     &phone,
		Role:      user.RoleCustomer,
		AvatarURL: &avatar,
		CreatedAt: now,
	}

	t.Run("success returns user", func(t *testing.T) {
		repo := &mockUserRepo{
			user: expectedUser,
		}
		exec := &mockExecutor{}

		svc := user.NewService(exec, nil, nil, nil, nil, repo)
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

		svc := user.NewService(exec, nil, nil, nil, nil, repo)
		result, err := svc.GetUserByID(ctx, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve user")
		assert.Equal(t, &user.User{}, result)
		assert.Equal(t, 1, repo.getByIDCalls)
	})
}
