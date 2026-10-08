package usecase

import (
	"context"
	"errors"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type mockUpdateShopAddressRepository struct {
	repository.ShopAddressRepository
	address          *domain.ShopAddress
	getErr           error
	updateCalls      int
	updateErr        error
	unsetActiveCalls int
	unsetActiveErr   error
}

func (m *mockUpdateShopAddressRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.ShopAddress, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.address, nil
}

func (m *mockUpdateShopAddressRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	address domain.ShopAddress,
) error {
	m.updateCalls++
	return m.updateErr
}

func (m *mockUpdateShopAddressRepository) UnsetActiveByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) error {
	m.unsetActiveCalls++
	return m.unsetActiveErr
}

type mockTransactor struct {
	transaction.Transactor
}

func (m *mockTransactor) WithinTransaction(
	ctx context.Context,
	fn func(exec transaction.Executor) error,
) error {
	return fn(&mockExecutor{})
}

type mockExecutor struct {
	transaction.Executor
}

func TestUpdateShopAddress_Success(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	existing := &domain.ShopAddress{
		ID:       addressID,
		ShopID:   shopID,
		Label:    "Old Label",
		IsActive: false,
	}

	repo := &mockUpdateShopAddressRepository{
		address: existing,
	}
	exec := &mockExecutor{}
	transactor := &mockTransactor{}

	isActive := true
	phone := "08123456789"
	input := UpdateShopAddressInput{
		ID:          addressID,
		ShopID:      shopID,
		Label:       "New Label",
		Phone:       &phone,
		IsActive:    &isActive,
		ProvinceID:  "32",
		CityID:      "3204",
		DistrictID:  "320401",
		VillageID:   "3204010001",
		FullAddress: "Jl. Baru No. 123",
		PostalCode:  "17520",
	}

	uc := NewAddressService(nil, repo, exec, transactor)

	err := uc.UpdateShopAddress(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.unsetActiveCalls != 1 {
		t.Errorf("expected UnsetActiveByShopID to be called 1 time, got %d", repo.unsetActiveCalls)
	}
	if repo.updateCalls != 1 {
		t.Errorf("expected Update to be called 1 time, got %d", repo.updateCalls)
	}
}

func TestUpdateShopAddress_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockUpdateShopAddressRepository{
		address: nil,
	}
	exec := &mockExecutor{}
	transactor := &mockTransactor{}

	input := UpdateShopAddressInput{
		ID:     uuid.New(),
		ShopID: uuid.New(),
	}

	uc := NewAddressService(nil, repo, exec, transactor)

	err := uc.UpdateShopAddress(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateShopAddress_ShopIDMismatch(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()
	differentShopID := uuid.New()

	existing := &domain.ShopAddress{
		ID:     addressID,
		ShopID: shopID,
	}

	repo := &mockUpdateShopAddressRepository{
		address: existing,
	}
	exec := &mockExecutor{}
	transactor := &mockTransactor{}

	input := UpdateShopAddressInput{
		ID:     addressID,
		ShopID: differentShopID,
	}

	uc := NewAddressService(nil, repo, exec, transactor)

	err := uc.UpdateShopAddress(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateShopAddress_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	// Soft-deleted address returns nil from GetByID
	repo := &mockUpdateShopAddressRepository{
		address: nil,
	}
	exec := &mockExecutor{}
	transactor := &mockTransactor{}

	input := UpdateShopAddressInput{
		ID:     addressID,
		ShopID: shopID,
	}

	uc := NewAddressService(nil, repo, exec, transactor)

	err := uc.UpdateShopAddress(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateShopAddress_RepoGetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db error")
	repo := &mockUpdateShopAddressRepository{
		getErr: expectedErr,
	}
	exec := &mockExecutor{}
	transactor := &mockTransactor{}

	input := UpdateShopAddressInput{
		ID:     uuid.New(),
		ShopID: uuid.New(),
	}

	uc := NewAddressService(nil, repo, exec, transactor)

	err := uc.UpdateShopAddress(ctx, input)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}
