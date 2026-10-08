package usecase

import (
	"context"
	"errors"
	"testing"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/domain"
	"komecore/internal/modules/inventory/repository"

	"github.com/google/uuid"
)

type mockCreateInventoryRepository struct {
	repository.InventoryRepository
	existing    *domain.Inventory
	created     *domain.Inventory
	getErr      error
	createErr   error
}

func (m *mockCreateInventoryRepository) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID, shopID uuid.UUID,
) (*domain.Inventory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.existing, nil
}

func (m *mockCreateInventoryRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	inv *domain.Inventory,
) error {
	m.created = inv
	return m.createErr
}

type mockProductChecker struct {
	exists bool
	err    error
}

func (m *mockProductChecker) ProductExists(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (bool, error) {
	return m.exists, m.err
}

type mockShopChecker struct {
	exists bool
	err    error
}

func (m *mockShopChecker) ShopExists(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func TestCreateInventory_Success(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()

	repo := &mockCreateInventoryRepository{}
	productChecker := &mockProductChecker{exists: true}
	shopChecker := &mockShopChecker{exists: true}
	exec := &mockExecutor{}
	recorder := &mockStockHistoryRecorder{}

	uc := NewCreateInventoryUsecase(repo, productChecker, shopChecker, exec, recorder)

	err := uc.Execute(ctx, CreateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     15,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.created == nil {
		t.Fatal("expected inventory to be created, got nil")
	}
	if repo.created.TotalStock != 15 {
		t.Errorf("expected TotalStock to be 15, got %d", repo.created.TotalStock)
	}
}

func TestCreateInventory_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockCreateInventoryRepository{}
	productChecker := &mockProductChecker{exists: false}
	shopChecker := &mockShopChecker{exists: true}
	exec := &mockExecutor{}
	recorder := &mockStockHistoryRecorder{}

	uc := NewCreateInventoryUsecase(repo, productChecker, shopChecker, exec, recorder)

	err := uc.Execute(ctx, CreateInventoryInput{
		ProductID: uuid.New(),
		ShopID:    uuid.New(),
		Stock:     15,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeNotFound {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestCreateInventory_ShopNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockCreateInventoryRepository{}
	productChecker := &mockProductChecker{exists: true}
	shopChecker := &mockShopChecker{exists: false}
	exec := &mockExecutor{}
	recorder := &mockStockHistoryRecorder{}

	uc := NewCreateInventoryUsecase(repo, productChecker, shopChecker, exec, recorder)

	err := uc.Execute(ctx, CreateInventoryInput{
		ProductID: uuid.New(),
		ShopID:    uuid.New(),
		Stock:     15,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeNotFound {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestCreateInventory_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	shopID := uuid.New()

	repo := &mockCreateInventoryRepository{
		existing: &domain.Inventory{
			ID:        uuid.New(),
			ProductID: productID,
			ShopID:    shopID,
		},
	}
	productChecker := &mockProductChecker{exists: true}
	shopChecker := &mockShopChecker{exists: true}
	exec := &mockExecutor{}
	recorder := &mockStockHistoryRecorder{}

	uc := NewCreateInventoryUsecase(repo, productChecker, shopChecker, exec, recorder)

	err := uc.Execute(ctx, CreateInventoryInput{
		ProductID: productID,
		ShopID:    shopID,
		Stock:     15,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeConflict {
		t.Errorf("expected Conflict error, got %v", err)
	}
}
