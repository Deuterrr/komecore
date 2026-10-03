package usecase

import (
	"context"
	"errors"
	"testing"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/domain"
	"komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepository "komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type mockDeleteInventoryRepository struct {
	repository.InventoryRepository
	inventory   *domain.Inventory
	getErr      error
	deleteErr   error
	deleteCalls int
}

type mockProductStockHistoryRepository struct {
	productRepository.ProductStockHistoryRepository
	recordedEvent *productDomain.ProductStockEvent
	recordErr     error
}

func (m *mockProductStockHistoryRepository) RecordStockEvent(
	ctx context.Context,
	exec transaction.Executor,
	event productDomain.ProductStockEvent,
) error {
	m.recordedEvent = &event
	return m.recordErr
}

func (m *mockDeleteInventoryRepository) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
) (*domain.Inventory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.inventory, nil
}

func (m *mockDeleteInventoryRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
) error {
	m.deleteCalls++
	return m.deleteErr
}

func TestDeleteInventory_Success(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &domain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 0,
	}

	repo := &mockDeleteInventoryRepository{
		inventory: existing,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockProductStockHistoryRepository{}

	uc := NewDeleteInventoryUsecase(repo, exec, stockHistoryRepo)

	err := uc.Execute(ctx, DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.deleteCalls != 1 {
		t.Errorf("expected Delete to be called 1 time, got %d", repo.deleteCalls)
	}
}

func TestDeleteInventory_NotFound(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()

	repo := &mockDeleteInventoryRepository{
		inventory: nil,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockProductStockHistoryRepository{}

	uc := NewDeleteInventoryUsecase(repo, exec, stockHistoryRepo)

	err := uc.Execute(ctx, DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeNotFound {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestDeleteInventory_ConflictWithReservations(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &domain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 3,
	}

	repo := &mockDeleteInventoryRepository{
		inventory: existing,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockProductStockHistoryRepository{}

	uc := NewDeleteInventoryUsecase(repo, exec, stockHistoryRepo)

	err := uc.Execute(ctx, DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeConflict {
		t.Errorf("expected Conflict error, got %v", err)
	}

	if repo.deleteCalls != 0 {
		t.Errorf("expected Delete to not be called, got %d calls", repo.deleteCalls)
	}
}

func TestDeleteInventory_RepoError(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()
	invID := uuid.New()

	existing := &domain.Inventory{
		ID:            invID,
		ProductID:     productID,
		ShopID:        shopID,
		TotalStock:    10,
		ReservedStock: 0,
	}

	expectedErr := errors.New("db delete error")
	repo := &mockDeleteInventoryRepository{
		inventory: existing,
		deleteErr: expectedErr,
	}
	exec := &mockExecutor{}
	stockHistoryRepo := &mockProductStockHistoryRepository{}

	uc := NewDeleteInventoryUsecase(repo, exec, stockHistoryRepo)

	err := uc.Execute(ctx, DeleteInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
