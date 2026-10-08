package addressusecase_test

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/modules/address/addressdomain"
	"komecore/internal/modules/address/addressusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListShopAddressesUsecase_FindByShopID(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()

	t.Run("success returns shop addresses", func(t *testing.T) {
		expected := []addressdomain.ShopAddress{
			{
				ID:       uuid.New(),
				ShopID:   shopID,
				Label:    "Branch 1",
				IsActive: true,
			},
		}
		repo := &mockShopAddressRepo{
			shopAddresses: expected,
		}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(nil, repo, exec, nil)
		result, err := uc.ListShopAddresses(ctx, shopID)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Equal(t, 1, repo.findByShopIDCalls)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := &mockShopAddressRepo{
			findByShopIDError: errors.New("db error"),
		}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(nil, repo, exec, nil)
		result, err := uc.ListShopAddresses(ctx, shopID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to retrieve address")
		assert.Equal(t, 1, repo.findByShopIDCalls)
	})
}
