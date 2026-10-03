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
	shopRepo "komecore/internal/modules/shop/repository"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

type CreateInventoryUsecase struct {
	inventoryRepo    repository.InventoryRepository
	productRepo      productRepository.ProductRepository
	shopRepo         shopRepo.ShopRepository
	executor         transaction.Executor
	stockHistoryRepo productRepository.ProductStockHistoryRepository
}

func NewCreateInventoryUsecase(
	inventoryRepo repository.InventoryRepository,
	productRepo productRepository.ProductRepository,
	shopRepo shopRepo.ShopRepository,
	executor transaction.Executor,
	stockHistoryRepo productRepository.ProductStockHistoryRepository,
) *CreateInventoryUsecase {
	return &CreateInventoryUsecase{
		inventoryRepo:    inventoryRepo,
		productRepo:      productRepo,
		shopRepo:         shopRepo,
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
	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to load product: %w", err)
	}
	if product == nil {
		return apperrors.NewNotFound(productDomain.ErrProductNotFound.Error())
	}

	shop, err := u.shopRepo.GetByID(ctx, u.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load shop: %w", err)
	}
	if shop == nil {
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
		event := productDomain.ProductStockEvent{
			ProductID: inventory.ProductID,
			ShopID:    inventory.ShopID,
			Available: inventory.TotalStock - inventory.ReservedStock,
		}

		_ = u.stockHistoryRepo.RecordStockEvent(context.Background(), u.executor,
			event,
		)
	}()

	return nil
}
