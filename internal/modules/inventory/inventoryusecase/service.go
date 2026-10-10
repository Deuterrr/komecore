package inventoryusecase

import (
	"context"
	"errors"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type CreateInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Stock     int
}

type UpdateInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Stock     int
}

type DeleteInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
}

type InventoryService struct {
	inventoryRepo    inventoryrepo.InventoryRepository
	productChecker   ProductChecker
	shopChecker      ShopChecker
	executor         transaction.Executor
	stockHistoryRepo StockHistoryRecorder
}

func NewInventoryService(
	inventoryRepo inventoryrepo.InventoryRepository,
	productChecker ProductChecker,
	shopChecker ShopChecker,
	executor transaction.Executor,
	stockHistoryRepo StockHistoryRecorder,
) *InventoryService {
	return &InventoryService{
		inventoryRepo:    inventoryRepo,
		productChecker:   productChecker,
		shopChecker:      shopChecker,
		executor:         executor,
		stockHistoryRepo: stockHistoryRepo,
	}
}

// CreateInventory verifies product and shop existence and initializes inventory stock.
func (s *InventoryService) CreateInventory(ctx context.Context, input CreateInventoryInput) error {
	productExists, err := s.productChecker.ProductExists(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check product existence: %w", err)
	}
	if !productExists {
		return apperror.NewNotFound("product not found")
	}

	shopExists, err := s.shopChecker.ShopExists(ctx, s.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to check shop existence: %w", err)
	}
	if !shopExists {
		return apperror.NewNotFound("shop not found")
	}

	existing, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}
	if existing != nil {
		return apperror.NewConflict("inventory already exists for product and shop")
	}

	inventory := &inventorydomain.Inventory{
		ID:            uuid.New(),
		ProductID:     input.ProductID,
		ShopID:        input.ShopID,
		TotalStock:    input.Stock,
		ReservedStock: 0,
		CreatedAt:     appclock.Now(),
	}
	if err := inventory.Validate(); err != nil {
		if errors.Is(err, inventorydomain.ErrInvalidStock) || errors.Is(err, inventorydomain.ErrInvalidReserved) {
			return apperror.NewInvalidInput(err.Error())
		}
		return err
	}

	if err := s.inventoryRepo.Create(ctx, s.executor, inventory); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	go func() {
		_ = s.stockHistoryRepo.RecordStockEvent(
			context.Background(),
			s.executor,
			inventory.ProductID,
			inventory.ShopID,
			inventory.TotalStock-inventory.ReservedStock,
		)
	}()

	return nil
}

// UpdateInventory updates the total stock for a given product and shop.
func (s *InventoryService) UpdateInventory(ctx context.Context, input UpdateInventoryInput) error {
	existing, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}
	if existing == nil {
		return apperror.NewNotFound("inventory not found")
	}

	existing.TotalStock = input.Stock

	if err := existing.Validate(); err != nil {
		if errors.Is(err, inventorydomain.ErrInvalidStock) ||
			errors.Is(err, inventorydomain.ErrInvalidReserved) ||
			errors.Is(err, inventorydomain.ErrReservedExceedsStock) {
			return apperror.NewInvalidInput(err.Error())
		}
		return err
	}

	if err := s.inventoryRepo.Update(ctx, s.executor, existing); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	go func() {
		_ = s.stockHistoryRepo.RecordStockEvent(
			context.Background(),
			s.executor,
			existing.ProductID,
			existing.ShopID,
			existing.TotalStock-existing.ReservedStock,
		)
	}()

	return nil
}

// DeleteInventory deletes the inventory record if there are no active reservations.
func (s *InventoryService) DeleteInventory(ctx context.Context, input DeleteInventoryInput) error {
	existing, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}
	if existing == nil {
		return apperror.NewNotFound("inventory not found")
	}
	if existing.ReservedStock > 0 {
		return apperror.NewConflict("cannot delete inventory with active reservations")
	}

	if err := s.inventoryRepo.Delete(ctx, s.executor, input.ProductID, input.ShopID); err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	go func() {
		_ = s.stockHistoryRepo.RecordStockEvent(
			context.Background(),
			s.executor,
			input.ProductID,
			input.ShopID,
			0,
		)
	}()

	return nil
}
