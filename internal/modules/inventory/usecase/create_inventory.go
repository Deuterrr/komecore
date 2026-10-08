package usecase

import (
	"context"
	"errors"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/domain"
	"komecore/internal/modules/inventory/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type CreateInventoryUsecase struct {
	inventoryRepo    repository.InventoryRepository
	productChecker   ProductChecker
	shopChecker      ShopChecker
	executor         transaction.Executor
	stockHistoryRepo StockHistoryRecorder
}

func NewCreateInventoryUsecase(
	inventoryRepo repository.InventoryRepository,
	productChecker ProductChecker,
	shopChecker ShopChecker,
	executor transaction.Executor,
	stockHistoryRepo StockHistoryRecorder,
) *CreateInventoryUsecase {
	return &CreateInventoryUsecase{
		inventoryRepo:    inventoryRepo,
		productChecker:   productChecker,
		shopChecker:      shopChecker,
		executor:         executor,
		stockHistoryRepo: stockHistoryRepo,
	}
}

type CreateInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Stock     int
}

func (u *CreateInventoryUsecase) Execute(ctx context.Context, input CreateInventoryInput) error {
	productExists, err := u.productChecker.ProductExists(ctx, u.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check product existence: %w", err)
	}
	if !productExists {
		return apperrors.NewNotFound("product not found")
	}

	shopExists, err := u.shopChecker.ShopExists(ctx, u.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to check shop existence: %w", err)
	}
	if !shopExists {
		return apperrors.NewNotFound("shop not found")
	}

	existing, err := u.inventoryRepo.GetByProductIDAndShopID(ctx, u.executor,
		input.ProductID,
		input.ShopID,
	)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}
	if existing != nil {
		return apperrors.NewConflict("inventory already exists for product and shop")
	}

	inventory := &domain.Inventory{
		ID:            uuid.New(),
		ProductID:     input.ProductID,
		ShopID:        input.ShopID,
		TotalStock:    input.Stock,
		ReservedStock: 0,
		CreatedAt:     appclock.Now(),
	}
	if err := inventory.Validate(); err != nil {
		if errors.Is(err, domain.ErrInvalidStock) || errors.Is(err, domain.ErrInvalidReserved) {
			return apperrors.NewInvalidInput(err.Error())
		}
		return err
	}

	if err := u.inventoryRepo.Create(ctx, u.executor, inventory); err != nil {
		return fmt.Errorf("failed to save inventory: %w", err)
	}

	go func() {
		_ = u.stockHistoryRepo.RecordStockEvent(
			context.Background(),
			u.executor,
			inventory.ProductID,
			inventory.ShopID,
			inventory.TotalStock-inventory.ReservedStock,
		)
	}()

	return nil
}
