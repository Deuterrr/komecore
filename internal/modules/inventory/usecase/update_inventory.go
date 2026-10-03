package usecase

import (
	"context"
	"errors"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/domain"
	"komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepository "komecore/internal/modules/product/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type UpdateInventoryUsecase struct {
	inventoryRepo    repository.InventoryRepository
	executor         transaction.Executor
	stockHistoryRepo productRepository.ProductStockHistoryRepository
}

func NewUpdateInventoryUsecase(
	inventoryRepo repository.InventoryRepository,
	executor transaction.Executor,
	stockHistoryRepo productRepository.ProductStockHistoryRepository,
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
		event := productDomain.ProductStockEvent{
			ProductID:  existing.ProductID,
			ShopID:     existing.ShopID,
			Available:  existing.TotalStock - existing.ReservedStock,
			RecordedAt: appclock.Now(),
		}

		_ = u.stockHistoryRepo.RecordStockEvent(context.Background(), u.executor,
			event,
		)
	}()

	return nil
}
