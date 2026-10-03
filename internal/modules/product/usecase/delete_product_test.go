package usecase

import (
	"context"
	"errors"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

// ===========================================================================
// Mocks for DeleteProductUsecase Tests
// ===========================================================================

type mockDeleteProductRepository struct {
	repository.ProductRepository
	product     *domain.Product
	getErr      error
	deleteErr   error
	deleteCalls int
}

func (m *mockDeleteProductRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.Product, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.product, nil
}

func (m *mockDeleteProductRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	m.deleteCalls++
	return m.deleteErr
}

// ===========================================================================
// Tests
// ===========================================================================

func TestDeleteProduct_Success(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	product := &domain.Product{
		ID:   productID,
		SKU:  "PROD-001",
		Name: "Test Product",
		Slug: "test-product",
	}

	repo := &mockDeleteProductRepository{
		product: product,
	}
	exec := &mockExecutor{}

	uc := NewDeleteProductUsecase(repo, exec)

	err := uc.Execute(ctx, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.deleteCalls != 1 {
		t.Errorf("expected Delete to be called 1 time, got %d", repo.deleteCalls)
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockDeleteProductRepository{
		product: nil,
	}
	exec := &mockExecutor{}

	uc := NewDeleteProductUsecase(repo, exec)

	err := uc.Execute(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteProduct_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()

	// In the persistence layer, already soft-deleted rows are excluded by
	// `WHERE deleted_at IS NULL`, returning nil from GetByID.
	repo := &mockDeleteProductRepository{
		product: nil,
	}
	exec := &mockExecutor{}

	uc := NewDeleteProductUsecase(repo, exec)

	err := uc.Execute(ctx, productID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if repo.deleteCalls != 0 {
		t.Errorf("expected Delete not to be called, but got %d calls", repo.deleteCalls)
	}
}

func TestDeleteProduct_RepoErrorOnGet(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db error on get")
	repo := &mockDeleteProductRepository{
		getErr: expectedErr,
	}
	exec := &mockExecutor{}

	uc := NewDeleteProductUsecase(repo, exec)

	err := uc.Execute(ctx, uuid.New())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestDeleteProduct_RepoErrorOnDelete(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	product := &domain.Product{
		ID:   productID,
		SKU:  "PROD-001",
		Name: "Test Product",
		Slug: "test-product",
	}

	expectedErr := errors.New("db error on delete")
	repo := &mockDeleteProductRepository{
		product:   product,
		deleteErr: expectedErr,
	}
	exec := &mockExecutor{}

	uc := NewDeleteProductUsecase(repo, exec)

	err := uc.Execute(ctx, productID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}
