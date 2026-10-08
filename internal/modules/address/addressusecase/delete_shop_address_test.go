package addressusecase

import (
	"context"
	"errors"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/addressdomain"

	"github.com/google/uuid"
)

type mockDeleteShopAddressRepository struct {
	address     *addressdomain.ShopAddress
	getErr      error
	deleteCalls int
	deleteErr   error
}

func (m *mockDeleteShopAddressRepository) FindByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) ([]addressdomain.ShopAddress, error) {
	return nil, nil
}

func (m *mockDeleteShopAddressRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*addressdomain.ShopAddress, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.address, nil
}

func (m *mockDeleteShopAddressRepository) UnsetActiveByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) error {
	return nil
}

func (m *mockDeleteShopAddressRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	address addressdomain.ShopAddress,
) error {
	return nil
}

func (m *mockDeleteShopAddressRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	address addressdomain.ShopAddress,
) error {
	return nil
}

func (m *mockDeleteShopAddressRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	m.deleteCalls++
	return m.deleteErr
}

func TestDeleteShopAddress_Success(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	existing := &addressdomain.ShopAddress{
		ID:       addressID,
		ShopID:   shopID,
		IsActive: false,
	}

	repo := &mockDeleteShopAddressRepository{
		address: existing,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, shopID, addressID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.deleteCalls != 1 {
		t.Errorf("expected Delete to be called 1 time, got %d", repo.deleteCalls)
	}
}

func TestDeleteShopAddress_ConflictWhenActive(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	existing := &addressdomain.ShopAddress{
		ID:       addressID,
		ShopID:   shopID,
		IsActive: true,
	}

	repo := &mockDeleteShopAddressRepository{
		address: existing,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, shopID, addressID)
	if err == nil {
		t.Fatal("expected conflict error when deleting active address, got nil")
	}

	if repo.deleteCalls != 0 {
		t.Errorf("expected Delete not to be called, got %d calls", repo.deleteCalls)
	}
}

func TestDeleteShopAddress_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockDeleteShopAddressRepository{
		address: nil,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteShopAddress_ShopIDMismatch(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	existing := &addressdomain.ShopAddress{
		ID:       addressID,
		ShopID:   shopID,
		IsActive: false,
	}

	repo := &mockDeleteShopAddressRepository{
		address: existing,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, uuid.New(), addressID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteShopAddress_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	addressID := uuid.New()
	shopID := uuid.New()

	// Soft-deleted address is excluded by repo query and returns nil
	repo := &mockDeleteShopAddressRepository{
		address: nil,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, shopID, addressID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteShopAddress_RepoGetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db get error")
	repo := &mockDeleteShopAddressRepository{
		getErr: expectedErr,
	}
	exec := &mockExecutor{}

	uc := NewAddressService(nil, repo, exec, nil)

	err := uc.DeleteShopAddress(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}
