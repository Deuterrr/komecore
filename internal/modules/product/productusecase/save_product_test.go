package productusecase

import (
	"context"
	"errors"
	"testing"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"

	"github.com/google/uuid"
)

// ===========================================================================
// Mocks for SaveProductUsecase Tests
// ===========================================================================

type mockProductRepository struct {
	productrepo.ProductRepository
	savedProduct *productdomain.Product
	saveErr      error
}

func (m *mockProductRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	product *productdomain.Product,
) error {
	m.savedProduct = product
	return m.saveErr
}

type mockSlugGenerator struct{}

func (m *mockSlugGenerator) Generate(input string) string {
	return "mocked-" + input
}

type mockExecutor struct {
	transaction.Executor
}

type mockTransactor struct {
	exec transaction.Executor
}

func (m *mockTransactor) WithinTransaction(
	ctx context.Context,
	fn func(transaction.Executor) error,
) error {
	return fn(m.exec)
}

type mockProductPerformanceRepository struct {
	productrepo.ProductPerformanceRepository
	upsertCalled bool
	upsertPerf   productdomain.ProductPerformance
	upsertPrice  int64
	upsertErr    error
}

func (m *mockProductPerformanceRepository) UpsertPerformance(
	ctx context.Context,
	exec transaction.Executor,
	perf productdomain.ProductPerformance,
	basePrice int64,
) error {
	m.upsertCalled = true
	m.upsertPerf = perf
	m.upsertPrice = basePrice
	return m.upsertErr
}

// ===========================================================================
// Tests
// ===========================================================================

func TestSaveProduct_Success_NewProduct(t *testing.T) {
	ctx := context.Background()
	transactor := &mockTransactor{exec: &mockExecutor{}}
	repo := &mockProductRepository{}
	slugGen := &mockSlugGenerator{}
	perfRepo := &mockProductPerformanceRepository{}

	uc := NewSaveProductUsecase(transactor, repo, slugGen, perfRepo)

	desc := "New product description"
	weight := 1.5

	input := SaveProductInput{
		ID:          nil,
		SKU:         "PROD-001",
		Name:        "Test Product",
		Description: &desc,
		Status:      "active",
		Price:       10000,
		Weight:      &weight,
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedProduct == nil {
		t.Fatal("expected product to be saved, got nil")
	}

	if repo.savedProduct.ID == uuid.Nil {
		t.Error("expected new UUID to be generated, got Nil")
	}

	if repo.savedProduct.SKU != "PROD-001" {
		t.Errorf("expected SKU 'PROD-001', got '%s'", repo.savedProduct.SKU)
	}

	if repo.savedProduct.Name != "Test Product" {
		t.Errorf("expected Name 'Test Product', got '%s'", repo.savedProduct.Name)
	}

	if repo.savedProduct.Slug != "mocked-Test Product" {
		t.Errorf("expected Slug 'mocked-Test Product', got '%s'", repo.savedProduct.Slug)
	}

	if *repo.savedProduct.Description != desc {
		t.Errorf("expected Description '%s', got '%s'", desc, *repo.savedProduct.Description)
	}

	if repo.savedProduct.Status != productdomain.ProductStatusActive {
		t.Errorf("expected Status 'active', got '%s'", repo.savedProduct.Status)
	}

	if repo.savedProduct.Price != 10000 {
		t.Errorf("expected Price 10000, got %d", repo.savedProduct.Price)
	}

	if *repo.savedProduct.Weight != weight {
		t.Errorf("expected Weight %v, got %v", weight, *repo.savedProduct.Weight)
	}
}

func TestSaveProduct_Success_UpdateProduct(t *testing.T) {
	ctx := context.Background()
	transactor := &mockTransactor{exec: &mockExecutor{}}
	repo := &mockProductRepository{}
	slugGen := &mockSlugGenerator{}
	perfRepo := &mockProductPerformanceRepository{}

	uc := NewSaveProductUsecase(transactor, repo, slugGen, perfRepo)

	productID := uuid.New()
	desc := "Updated description"
	weight := 2.0

	input := SaveProductInput{
		ID:          &productID,
		SKU:         "PROD-001-UPD",
		Name:        "Updated Product",
		Description: &desc,
		Status:      "inactive",
		Price:       15000,
		Weight:      &weight,
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedProduct == nil {
		t.Fatal("expected product to be saved, got nil")
	}

	if repo.savedProduct.ID != productID {
		t.Errorf("expected product ID %v, got %v", productID, repo.savedProduct.ID)
	}

	if repo.savedProduct.SKU != "PROD-001-UPD" {
		t.Errorf("expected SKU 'PROD-001-UPD', got '%s'", repo.savedProduct.SKU)
	}

	if repo.savedProduct.Name != "Updated Product" {
		t.Errorf("expected Name 'Updated Product', got '%s'", repo.savedProduct.Name)
	}

	if repo.savedProduct.Slug != "mocked-Updated Product" {
		t.Errorf("expected Slug 'mocked-Updated Product', got '%s'", repo.savedProduct.Slug)
	}

	if *repo.savedProduct.Description != desc {
		t.Errorf("expected Description '%s', got '%s'", desc, *repo.savedProduct.Description)
	}

	if repo.savedProduct.Status != productdomain.ProductStatusInactive {
		t.Errorf("expected Status 'inactive', got '%s'", repo.savedProduct.Status)
	}

	if repo.savedProduct.Price != 15000 {
		t.Errorf("expected Price 15000, got %d", repo.savedProduct.Price)
	}

	if *repo.savedProduct.Weight != weight {
		t.Errorf("expected Weight %v, got %v", weight, *repo.savedProduct.Weight)
	}
}

func TestSaveProduct_InvalidInput_EmptyName(t *testing.T) {
	ctx := context.Background()
	transactor := &mockTransactor{exec: &mockExecutor{}}
	repo := &mockProductRepository{}
	slugGen := &mockSlugGenerator{}
	perfRepo := &mockProductPerformanceRepository{}

	uc := NewSaveProductUsecase(transactor, repo, slugGen, perfRepo)

	input := SaveProductInput{
		ID:     nil,
		SKU:    "PROD-001",
		Name:   "",
		Status: "active",
		Price:  10000,
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSaveProduct_InvalidInput_NegativePrice(t *testing.T) {
	ctx := context.Background()
	transactor := &mockTransactor{exec: &mockExecutor{}}
	repo := &mockProductRepository{}
	slugGen := &mockSlugGenerator{}
	perfRepo := &mockProductPerformanceRepository{}

	uc := NewSaveProductUsecase(transactor, repo, slugGen, perfRepo)

	input := SaveProductInput{
		ID:     nil,
		SKU:    "PROD-001",
		Name:   "Test Product",
		Status: "active",
		Price:  -50,
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSaveProduct_RepoError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("database connection failed")
	transactor := &mockTransactor{exec: &mockExecutor{}}
	repo := &mockProductRepository{saveErr: expectedErr}
	slugGen := &mockSlugGenerator{}
	perfRepo := &mockProductPerformanceRepository{}

	uc := NewSaveProductUsecase(transactor, repo, slugGen, perfRepo)

	input := SaveProductInput{
		ID:     nil,
		SKU:    "PROD-001",
		Name:   "Test Product",
		Status: "active",
		Price:  10000,
	}

	err := uc.Execute(ctx, input)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}
