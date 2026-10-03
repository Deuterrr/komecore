package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepository "komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type DeleteInventoryUsecase struct {
	inventoryRepo    repository.InventoryRepository
	executor         transaction.Executor
	stockHistoryRepo productRepository.ProductStockHistoryRepository
}

func NewDeleteInventoryUsecase(
	inventoryRepo repository.InventoryRepository,
	executor transaction.Executor,
	stockHistoryRepo productRepository.ProductStockHistoryRepository,
) *DeleteInventoryUsecase {
	return &DeleteInventoryUsecase{
		inventoryRepo:    inventoryRepo,
		executor:         executor,
		stockHistoryRepo: stockHistoryRepo,
	}
}

type DeleteInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
}

func (u *DeleteInventoryUsecase) Execute(ctx context.Context, input DeleteInventoryInput) error {
	existing, err := u.inventoryRepo.GetByProductIDAndShopID(ctx, u.executor,
		input.ProductID,
		input.ShopID,
	)
	if err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}
	if existing == nil {
		return apperrors.NewNotFound("inventory not found")
	}
	if existing.ReservedStock > 0 {
		return apperrors.NewConflict("cannot delete inventory with active reservations")
	}

	if err := u.inventoryRepo.Delete(ctx, u.executor, input.ProductID, input.ShopID); err != nil {
		return fmt.Errorf("failed to delete inventory: %w", err)
	}

	go func() {
		_ = u.stockHistoryRepo.RecordStockEvent(context.Background(), u.executor,
			productDomain.ProductStockEvent{
				ProductID: input.ProductID,
				ShopID:    input.ShopID,
				Available: 0,
			},
		)
	}()

	return nil
}
