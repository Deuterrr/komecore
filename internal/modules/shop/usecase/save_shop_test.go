package usecase

import (
	"context"
	"testing"

	transaction "komecore/internal/infra/transactor"
	authorDomain "komecore/internal/modules/auth/domain"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"
	slug "komecore/pkg/slug"

	"github.com/google/uuid"
)

type mockSaveShopRepository struct {
	repository.ShopRepository
	savedShop *domain.Shop
	shopByID  *domain.Shop
	getErr    error
	saveErr   error
}

func (m *mockSaveShopRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.Shop, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.shopByID, nil
}

func (m *mockSaveShopRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	shop domain.Shop,
) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedShop = &shop
	return nil
}

func TestSaveShop_AdminCreate(t *testing.T) {
	ctx := context.Background()
	repo := &mockSaveShopRepository{}
	exec := &mockExecutor{}
	slugGen := slug.NewGenerator()

	actor := authorDomain.Actor{
		Roles: []authorDomain.Role{
			{Code: authorDomain.RoleStaffAdmin},
		},
	}

	uc := NewSaveShopUsecase(repo, slugGen, exec)

	isActive := true
	approvalStatus := string(domain.ShopApprovalStatusApproved)
	desc := "Admin Shop"

	err := uc.Execute(ctx, actor, SaveShopInput{
		Name:           "Jakarta Central",
		Description:    &desc,
		IsActive:       &isActive,
		ApprovalStatus: &approvalStatus,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedShop == nil {
		t.Fatal("expected shop to be saved")
	}

	if !repo.savedShop.IsActive {
		t.Errorf("expected IsActive to be true, got %v", repo.savedShop.IsActive)
	}

	if repo.savedShop.ApprovalStatus != domain.ShopApprovalStatusApproved {
		t.Errorf("expected ApprovalStatus to be approved, got %v", repo.savedShop.ApprovalStatus)
	}
}

func TestSaveShop_RegularStaffCreate(t *testing.T) {
	ctx := context.Background()
	repo := &mockSaveShopRepository{}
	exec := &mockExecutor{}
	slugGen := slug.NewGenerator()

	actor := authorDomain.Actor{
		Roles: []authorDomain.Role{
			{Code: authorDomain.RoleStaff},
		},
	}

	uc := NewSaveShopUsecase(repo, slugGen, exec)

	isActive := true
	approvalStatus := string(domain.ShopApprovalStatusApproved)
	desc := "Staff Shop"

	// Regular staff attempts to pass active=true and approved
	err := uc.Execute(ctx, actor, SaveShopInput{
		Name:           "Bandung Branch",
		Description:    &desc,
		IsActive:       &isActive,
		ApprovalStatus: &approvalStatus,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedShop == nil {
		t.Fatal("expected shop to be saved")
	}

	// Should be forced to inactive and pending
	if repo.savedShop.IsActive {
		t.Errorf("expected IsActive to be false for regular staff, got %v", repo.savedShop.IsActive)
	}

	if repo.savedShop.ApprovalStatus != domain.ShopApprovalStatusPending {
		t.Errorf("expected ApprovalStatus to be pending for regular staff, got %v", repo.savedShop.ApprovalStatus)
	}
}

func TestSaveShop_RegularStaffUpdatePreservesStatus(t *testing.T) {
	ctx := context.Background()
	shopID := uuid.New()
	existingShop := &domain.Shop{
		ID:             shopID,
		Name:           "Old Name",
		Slug:           "old-name",
		IsActive:       true,
		ApprovalStatus: domain.ShopApprovalStatusApproved,
	}

	repo := &mockSaveShopRepository{
		shopByID: existingShop,
	}
	exec := &mockExecutor{}
	slugGen := slug.NewGenerator()

	actor := authorDomain.Actor{
		Roles: []authorDomain.Role{
			{Code: authorDomain.RoleStaff},
		},
	}

	uc := NewSaveShopUsecase(repo, slugGen, exec)

	newDesc := "Updated Description"
	err := uc.Execute(ctx, actor, SaveShopInput{
		ID:          &shopID,
		Name:        "New Name",
		Description: &newDesc,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.savedShop.Name != "New Name" {
		t.Errorf("expected name to be updated, got %s", repo.savedShop.Name)
	}

	if !repo.savedShop.IsActive {
		t.Errorf("expected existing IsActive (true) to be preserved, got %v", repo.savedShop.IsActive)
	}

	if repo.savedShop.ApprovalStatus != domain.ShopApprovalStatusApproved {
		t.Errorf("expected existing ApprovalStatus (approved) to be preserved, got %v", repo.savedShop.ApprovalStatus)
	}
}
