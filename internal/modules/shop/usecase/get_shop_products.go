package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type GetShopProductsUsecase struct {
	inventoryRepo inventoryRepo.InventoryRepository
	productRepo   productRepo.ProductRepository
	executor      transaction.Executor
}

func NewGetShopProductsUsecase(
	inventoryRepo inventoryRepo.InventoryRepository,
	productRepo productRepo.ProductRepository,
	executor transaction.Executor,
) *GetShopProductsUsecase {
	return &GetShopProductsUsecase{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		executor:      executor,
	}
}

// ShopProductResult pairs a product with its inventory record for a given shop.
type ShopProductResult struct {
	Product   productDomain.Product
	Inventory inventoryDomain.Inventory
}

func (u *GetShopProductsUsecase) Execute(
	ctx context.Context,
	shopID uuid.UUID,
) ([]ShopProductResult, error) {
	inventories, err := u.inventoryRepo.ListByShopID(ctx, u.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop inventories: %w", err)
	}
	if len(inventories) == 0 {
		return []ShopProductResult{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(inventories))
	for _, inv := range inventories {
		productIDs = append(productIDs, inv.ProductID)
	}
	products, err := u.productRepo.FindByIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for shop: %w", err)
	}
	productMap := make(map[uuid.UUID]productDomain.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	results := make([]ShopProductResult, 0, len(inventories))
	for _, inv := range inventories {
		p, ok := productMap[inv.ProductID]
		if !ok {
			continue
		}
		results = append(results, ShopProductResult{
			Product:   p,
			Inventory: inv,
		})
	}

	return results, nil
}
