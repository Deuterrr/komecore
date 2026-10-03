package usecase_test

import (
	"context"
	"errors"
	"testing"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"
	"komecore/internal/modules/address/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockShopAddressRepo struct {
	repository.ShopAddressRepository
	createCalls       int
	createError       error
	savedAddress      *domain.ShopAddress
	findByShopIDCalls int
	findByShopIDError error
	shopAddresses     []domain.ShopAddress
}

func (m *mockShopAddressRepo) Create(ctx context.Context, exec transaction.Executor, address domain.ShopAddress) error {
	m.createCalls++
	m.savedAddress = &address
	return m.createError
}

func (m *mockShopAddressRepo) FindByShopID(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]domain.ShopAddress, error) {
	m.findByShopIDCalls++
	if m.findByShopIDError != nil {
		return nil, m.findByShopIDError
	}
	return m.shopAddresses, nil
}

func TestCreateShopAddressUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	phone := "+6281111111"
	lat := -6.2
	lng := 106.8
	isActive := true

	baseInput := usecase.CreateShopAddressInput{
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

		uc := usecase.NewCreateShopAddressUsecase(repo, exec)
		err := uc.Execute(ctx, baseInput)

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

		uc := usecase.NewCreateShopAddressUsecase(repo, exec)
		invalidLng := 200.0 // > 180
		input := baseInput
		input.Longitude = &invalidLng

		err := uc.Execute(ctx, input)
		assert.Error(t, err)
		assert.True(t, apperrors.IsBadRequest(err))
		assert.Equal(t, 0, repo.createCalls)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		repo := &mockShopAddressRepo{
			createError: errors.New("db error"),
		}
		exec := &mockExecutor{}

		uc := usecase.NewCreateShopAddressUsecase(repo, exec)
		err := uc.Execute(ctx, baseInput)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save address")
		assert.Equal(t, 1, repo.createCalls)
	})
}
