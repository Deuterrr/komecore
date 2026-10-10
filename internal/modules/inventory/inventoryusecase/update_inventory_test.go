package inventoryusecase

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"

	"github.com/google/uuid"
)

type mockUpdateInventoryRepository struct {
	inventoryrepo.InventoryRepository
	inventory *inventorydomain.Inventory
	getErr    error
	updateErr error
	updated   *inventorydomain.Inventory
}

func (m *mockUpdateInventoryRepository) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
) (*inventorydomain.Inventory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.inventory, nil
}

func (m *mockUpdateInventoryRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	inventory *inventorydomain.Inventory,
) error {
	m.updated = inventory
	return m.updateErr
}

type mockExecutor struct {
	transaction.Executor
}

func TestUpdateInventory_Success(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &inventorydomain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 2,
	}

	repo := &mockUpdateInventoryRepository{
		inventory: existing,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockStockHistoryRecorder{}

	uc := NewInventoryService(repo, nil, nil, exec, stockHistoryRepo)

	err := uc.UpdateInventory(ctx, UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     20,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updated == nil {
		t.Fatal("expected update to be called, but got nil")
	}

	if repo.updated.TotalStock != 20 {
		t.Errorf("expected TotalStock to be 20, got %d", repo.updated.TotalStock)
	}
}

func TestUpdateInventory_NotFound(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()

	repo := &mockUpdateInventoryRepository{
		inventory: nil,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockStockHistoryRecorder{}

	uc := NewInventoryService(repo, nil, nil, exec, stockHistoryRepo)

	err := uc.UpdateInventory(ctx, UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     20,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeNotFound {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestUpdateInventory_InvalidStock(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &inventorydomain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 2,
	}

	repo := &mockUpdateInventoryRepository{
		inventory: existing,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockStockHistoryRecorder{}

	uc := NewInventoryService(repo, nil, nil, exec, stockHistoryRepo)

	// Attempting to set stock to less than reserved stock
	err := uc.UpdateInventory(ctx, UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     1,
	})

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeInvalidInput {
		t.Errorf("expected InvalidInput error, got %v", err)
	}
}

func TestUpdateInventory_RepoError(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &inventorydomain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 2,
	}

	expectedErr := errors.New("db error")
	repo := &mockUpdateInventoryRepository{
		inventory: existing,
		updateErr: expectedErr,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockStockHistoryRecorder{}

	uc := NewInventoryService(repo, nil, nil, exec, stockHistoryRepo)

	err := uc.UpdateInventory(ctx, UpdateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     20,
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
