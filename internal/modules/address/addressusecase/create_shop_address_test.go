package addressusecase_test

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/addressdomain"
	"komecore/internal/modules/address/addressusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockShopAddressRepo struct {
	createCalls       int
	createError       error
	savedAddress      *addressdomain.ShopAddress
	findByShopIDCalls int
	findByShopIDError error
	shopAddresses     []addressdomain.ShopAddress
}

func (m *mockShopAddressRepo) Create(ctx context.Context, exec transaction.Executor, address addressdomain.ShopAddress) error {
	m.createCalls++
	m.savedAddress = &address
	return m.createError
}

func (m *mockShopAddressRepo) FindByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]addressdomain.ShopAddress, error) {
	m.findByShopIDCalls++
	if m.findByShopIDError != nil {
		return nil, m.findByShopIDError
	}
	return m.shopAddresses, nil
}

func (m *mockShopAddressRepo) GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*addressdomain.ShopAddress, error) {
	return nil, nil
}

func (m *mockShopAddressRepo) UnsetActiveByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) error {
	return nil
}

func (m *mockShopAddressRepo) Update(ctx context.Context, exec transaction.Executor, address addressdomain.ShopAddress) error {
	return nil
}

func (m *mockShopAddressRepo) Delete(ctx context.Context, exec transaction.Executor, addressID uuid.UUID) error {
	return nil
}

func TestCreateShopAddressUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	phone := "+6281111111"
	lat := -6.2
	lng := 106.8
	isActive := true

	baseInput := addressusecase.CreateShopAddressInput{
		ShopID:      shopID,
		Label:       "Main Warehouse",
		Phone:       &phone,
		IsActive:    &isActive,
		Province:    "Jawa Barat",
		City:        "Bandung",
		District:    "Coblong",
		FullAddress: "Jl. Dago No. 50",
		PostalCode:  "40132",
		Latitude:    &lat,
		Longitude:   &lng,
	}

	t.Run("success creates shop address", func(t *testing.T) {
		repo := &mockShopAddressRepo{}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(nil, repo, exec, nil)
		err := uc.CreateShopAddress(ctx, baseInput)

		require.NoError(t, err)
		assert.Equal(t, 1, repo.createCalls)
		require.NotNil(t, repo.savedAddress)
		assert.Equal(t, shopID, repo.savedAddress.ShopID)
		assert.Equal(t, "Main Warehouse", repo.savedAddress.Label)
		assert.True(t, repo.savedAddress.IsActive)
	})

	t.Run("returns bad request on invalid coordinates", func(t *testing.T) {
		repo := &mockShopAddressRepo{}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(nil, repo, exec, nil)
		invalidLng := 200.0 // > 180
		input := baseInput
		input.Longitude = &invalidLng

		err := uc.CreateShopAddress(ctx, input)
		assert.Error(t, err)
		assert.True(t, apperror.IsBadRequest(err))
		assert.Equal(t, 0, repo.createCalls)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		repo := &mockShopAddressRepo{
			createError: errors.New("db error"),
		}
		exec := &mockExecutor{}

		uc := addressusecase.NewAddressService(nil, repo, exec, nil)
		err := uc.CreateShopAddress(ctx, baseInput)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save address")
		assert.Equal(t, 1, repo.createCalls)
	})
}
