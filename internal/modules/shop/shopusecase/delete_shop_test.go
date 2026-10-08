package shopusecase

import (
	"context"
	"errors"
	"testing"

	"komecore/internal/common/authctx"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/shopdomain"
	"komecore/internal/modules/shop/shoprepo"

	"github.com/google/uuid"
)

type mockDeleteShopRepository struct {
	shoprepo.ShopRepository
	shop        *shopdomain.Shop
	getErr      error
	deleteCalls int
	deleteErr   error
}

func (m *mockDeleteShopRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*shopdomain.Shop, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.shop, nil
}

func (m *mockDeleteShopRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	m.deleteCalls++
	return m.deleteErr
}

type mockExecutor struct {
	transaction.Executor
}

func TestDeleteShop_Success(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	shop := &shopdomain.Shop{
		ID:   shopID,
		Name: "Test Shop",
		Slug: "test-shop",
	}

	repo := &mockDeleteShopRepository{
		shop: shop,
	}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, shopID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.deleteCalls != 1 {
		t.Errorf("expected Delete to be called 1 time, got %d", repo.deleteCalls)
	}
}

func TestDeleteShop_ForbiddenForNonAdmin(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	repo := &mockDeleteShopRepository{}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaff},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, shopID)
	if err == nil {
		t.Fatal("expected error for non-admin actor, got nil")
	}

	if repo.deleteCalls != 0 {
		t.Errorf("expected Delete not to be called, got %d calls", repo.deleteCalls)
	}
}

func TestDeleteShop_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockDeleteShopRepository{
		shop: nil,
	}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteShop_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()

	// In the persistence layer, already soft-deleted rows are excluded by
	// `WHERE deleted_at IS NULL`, returning nil from GetByID.
	repo := &mockDeleteShopRepository{
		shop: nil,
	}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, shopID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if repo.deleteCalls != 0 {
		t.Errorf("expected Delete not to be called, but got %d calls", repo.deleteCalls)
	}
}

func TestDeleteShop_RepoErrorOnGet(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db error on get")
	repo := &mockDeleteShopRepository{
		getErr: expectedErr,
	}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, uuid.New())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestDeleteShop_RepoErrorOnDelete(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	shop := &shopdomain.Shop{
		ID:   shopID,
		Name: "Test Shop",
		Slug: "test-shop",
	}

	expectedErr := errors.New("db error on delete")
	repo := &mockDeleteShopRepository{
		shop:      shop,
		deleteErr: expectedErr,
	}
	exec := &mockExecutor{}

	actor := authctx.Actor{
		Roles: []authctx.Role{
			{Code: authctx.RoleStaffAdmin},
		},
	}

	uc := NewShopService(repo, nil, nil, nil, exec)

	err := uc.DeleteShop(ctx, actor, shopID)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}
