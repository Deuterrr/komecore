package usecase

import (
	"context"
	"errors"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/domain"
	"komecore/internal/modules/inventory/repository"

	"github.com/google/uuid"
)

type UpdateInventoryUsecase struct {
	inventoryRepo    repository.InventoryRepository
	executor         transaction.Executor
	stockHistoryRepo StockHistoryRecorder
}

func NewUpdateInventoryUsecase(
	inventoryRepo repository.InventoryRepository,
	executor transaction.Executor,
	stockHistoryRepo StockHistoryRecorder,
) *UpdateInventoryUsecase {
	return &UpdateInventoryUsecase{
		inventoryRepo:    inventoryRepo,
		executor:         executor,
		stockHistoryRepo: stockHistoryRepo,
	}
}

type UpdateInventoryInput struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Stock     int
}

func (u *UpdateInventoryUsecase) Execute(
	ctx context.Context,
	input UpdateInventoryInput,
) error {
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

	existing.TotalStock = input.Stock

	if err := existing.Validate(); err != nil {
		if errors.Is(err, domain.ErrInvalidStock) ||
			errors.Is(err, domain.ErrInvalidReserved) ||
			errors.Is(err, domain.ErrReservedExceedsStock) {
			return apperrors.NewInvalidInput(err.Error())
		}
		return err
	}

	if err := u.inventoryRepo.Update(ctx, u.executor, existing); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	go func() {
		_ = u.stockHistoryRepo.RecordStockEvent(
			context.Background(),
			u.executor,
			existing.ProductID,
			existing.ShopID,
			existing.TotalStock-existing.ReservedStock,
		)
	}()

	return nil
}
