package usecase_test

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCustomerAddressesUsecase_ListByCustomerID(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	otherCustomerID := uuid.New()

	t.Run("success returns addresses for specified customer only", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		addr1 := domain.CustomerAddress{
			ID:           uuid.New(),
			CustomerID:   customerID,
			ReceiverName: "Customer 1 - Home",
			IsDefault:    true,
		}
		addr2 := domain.CustomerAddress{
			ID:           uuid.New(),
			CustomerID:   customerID,
			ReceiverName: "Customer 1 - Office",
			IsDefault:    false,
		}
		addrOther := domain.CustomerAddress{
			ID:           uuid.New(),
			CustomerID:   otherCustomerID,
			ReceiverName: "Other Customer Address",
			IsDefault:    true,
		}

		repo.addresses[addr1.ID] = addr1
		repo.addresses[addr2.ID] = addr2
		repo.addresses[addrOther.ID] = addrOther

		exec := &mockExecutor{}
		uc := usecase.NewAddressService(repo, nil, exec, nil)

		result, err := uc.ListCustomerAddresses(ctx, customerID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 1, repo.listCalls)
	})

	t.Run("success returns empty slice when customer has no addresses", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		exec := &mockExecutor{}
		uc := usecase.NewAddressService(repo, nil, exec, nil)

		result, err := uc.ListCustomerAddresses(ctx, customerID)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 1, repo.listCalls)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.listError = errors.New("db list error")
		exec := &mockExecutor{}
		uc := usecase.NewAddressService(repo, nil, exec, nil)

		result, err := uc.ListCustomerAddresses(ctx, customerID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to retrieve address")
		assert.Equal(t, 1, repo.listCalls)
	})
}
