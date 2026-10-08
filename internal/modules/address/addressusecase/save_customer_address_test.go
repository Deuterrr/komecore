package addressusecase_test

import (
	"context"
	"errors"
	"testing"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/address/addressusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveCustomerAddressUsecase_Execute(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	phone := "+628123456789"
	lat := -6.2088
	lng := 106.8456

	baseInput := addressusecase.SaveCustomerAddressInput{
		CustomerID:   customerID,
		ReceiverName: "Jane Doe",
		Phone:        &phone,
		Province:     "DKI Jakarta",
		City:         "Jakarta Selatan",
		District:     "Tebet",
		FullAddress:  "Jl. Tebet Barat No. 10",
		PostalCode:   "12810",
		Latitude:     &lat,
		Longitude:    &lng,
	}

	t.Run("success create first address automatically becomes default", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countZero := 0
		repo.count = &countZero
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput

		err := uc.SaveCustomerAddress(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.saveCalls)
		assert.Equal(t, 1, repo.unsetDefaultCalls)
		require.Len(t, repo.savedAddresses, 1)
		saved := repo.savedAddresses[0]
		assert.Equal(t, customerID, saved.CustomerID)
		assert.Equal(t, "Jane Doe", saved.ReceiverName)
		assert.True(t, saved.IsDefault)
	})

	t.Run("success create non-default address when already has addresses", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countTwo := 2
		repo.count = &countTwo
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		isDefaultFalse := false
		input := baseInput
		input.IsDefault = &isDefaultFalse

		err := uc.SaveCustomerAddress(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.saveCalls)
		assert.Equal(t, 0, repo.unsetDefaultCalls)
		require.Len(t, repo.savedAddresses, 1)
		assert.False(t, repo.savedAddresses[0].IsDefault)
	})

	t.Run("success create explicit default unsets prior defaults", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countTwo := 2
		repo.count = &countTwo
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		isDefaultTrue := true
		input := baseInput
		input.IsDefault = &isDefaultTrue

		err := uc.SaveCustomerAddress(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.saveCalls)
		assert.Equal(t, 1, repo.unsetDefaultCalls)
		require.Len(t, repo.savedAddresses, 1)
		assert.True(t, repo.savedAddresses[0].IsDefault)
	})

	t.Run("returns conflict when address limit is reached (>= 10)", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countTen := 10
		repo.count = &countTen
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput

		err := uc.SaveCustomerAddress(ctx, input)
		assert.Error(t, err)
		assert.True(t, apperrors.IsConflict(err))
		assert.Equal(t, 0, repo.saveCalls)
	})

	t.Run("returns bad request on invalid coordinates", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		invalidLat := 120.0 // > 90
		input := baseInput
		input.Latitude = &invalidLat

		err := uc.SaveCustomerAddress(ctx, input)
		assert.Error(t, err)
		assert.True(t, apperrors.IsBadRequest(err))
		assert.Equal(t, 0, repo.saveCalls)
	})

	t.Run("success update existing address", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		existingID := uuid.New()
		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput
		input.ID = &existingID
		isDefaultFalse := false
		input.IsDefault = &isDefaultFalse

		err := uc.SaveCustomerAddress(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.saveCalls)
		assert.Equal(t, 0, repo.countCalls) // count not called on update
		require.Len(t, repo.savedAddresses, 1)
		assert.Equal(t, existingID, repo.savedAddresses[0].ID)
	})

	t.Run("returns error when CountByCustomerID fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		repo.countError = errors.New("db count failed")
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput

		err := uc.SaveCustomerAddress(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count addresses")
		assert.Equal(t, 0, repo.saveCalls)
	})

	t.Run("returns error when UnsetDefaultByCustomerID fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countZero := 0
		repo.count = &countZero
		repo.unsetDefaultErr = errors.New("unset default failed")
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput

		err := uc.SaveCustomerAddress(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unset default address")
	})

	t.Run("returns error when Save fails", func(t *testing.T) {
		repo := newMockCustomerAddressRepo()
		countZero := 0
		repo.count = &countZero
		repo.saveError = errors.New("db save failed")
		exec := &mockExecutor{}
		tx := &mockTransactor{}

		uc := addressusecase.NewAddressService(repo, nil, exec, tx)
		input := baseInput

		err := uc.SaveCustomerAddress(ctx, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save address")
	})
}
