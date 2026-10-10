package addressusecase_test

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/address/addressdomain"
	"komecore/internal/modules/address/addressusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCustomerAddressUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	customerID := uuid.New()

	t.Run("success deletes non-default address", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.addresses[addressID] = addressdomain.CustomerAddress{
			ID:         addressID,
			CustomerID: customerID,
			IsDefault:  false,
		}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, nil)
		err := uc.DeleteCustomerAddress(ctx, addressID)

		require.NoError(t, err)
		assert.Equal(t, 1, repo.getByIDCalls)
		assert.Equal(t, 1, repo.deleteCalls)
		assert.NotContains(t, repo.addresses, addressID)
	})

	t.Run("returns not found when address does not exist", func(t *testing.T) {
		repo := newMockCustomerAddressRepo() // empty
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, nil)
		err := uc.DeleteCustomerAddress(ctx, addressID)

		assert.Error(t, err)
		assert.True(t, apperror.IsNotFound(err))
		assert.Equal(t, 0, repo.deleteCalls)
	})

	t.Run("returns conflict when trying to delete default address", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.addresses[addressID] = addressdomain.CustomerAddress{
			ID:         addressID,
			CustomerID: customerID,
			IsDefault:  true,
		}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, nil)
		err := uc.DeleteCustomerAddress(ctx, addressID)

		assert.Error(t, err)
		assert.True(t, apperror.IsConflict(err))
		assert.Equal(t, 0, repo.deleteCalls)
	})

	t.Run("returns error when GetByID fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.getByIDError = errors.New("db get error")
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, nil)
		err := uc.DeleteCustomerAddress(ctx, addressID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve address")
		assert.Equal(t, 0, repo.deleteCalls)
	})

	t.Run("returns error when Delete repository call fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.addresses[addressID] = addressdomain.CustomerAddress{
			ID:         addressID,
			CustomerID: customerID,
			IsDefault:  false,
		}
		repo.deleteError = errors.New("db delete error")
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, nil)
		err := uc.DeleteCustomerAddress(ctx, addressID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete address")
		assert.Equal(t, 1, repo.deleteCalls)
	})
}
