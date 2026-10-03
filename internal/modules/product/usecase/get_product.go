package usecase

import (
	"context"
	"fmt"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
	shopDomain "komecore/internal/modules/shop/domain"
	shopRepo "komecore/internal/modules/shop/repository"

	"github.com/google/uuid"
)

type GetProductUsecase struct {
	executor       transaction.Executor
	fileStore      storage.Provider
	productRepo    repository.ProductRepository
	inventoryRepo  inventoryRepo.InventoryRepository
	productImgRepo repository.ProductImageRepository
	shopRepo       shopRepo.ShopRepository
	perfRepo       repository.ProductPerformanceRepository
}

func NewGetProductUsecase(
	executor transaction.Executor,
	fileStore storage.Provider,
	productRepo repository.ProductRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productImgRepo repository.ProductImageRepository,
	shopRepo shopRepo.ShopRepository,
	perfRepo repository.ProductPerformanceRepository,
) *GetProductUsecase {
	return &GetProductUsecase{
		executor:       executor,
		fileStore:      fileStore,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		productImgRepo: productImgRepo,
		shopRepo:       shopRepo,
		perfRepo:       perfRepo,
	}
}

type ImageProductDetail struct {
	Thumbnail string
	Detail    string
	Preview   string
}

type ProductDetailResult struct {
	Product   domain.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventoryDomain.Inventory
	Images          []ImageProductDetail
	Availability    []ShopAvailabilityResult
}

func (u *GetProductUsecase) Execute(
	ctx context.Context,
	slug string,
) (*ProductDetailResult, error) {
	product, err := u.productRepo.GetBySlug(ctx, u.executor, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to load products with inventory: %w", err)
	}
	if product == nil {
		return nil, nil
	}

	go func() {
		_ = u.perfRepo.IncrementViewCount(context.Background(), u.executor, product.ID)
	}()

	inventories, err := u.inventoryRepo.ListByProductID(ctx, u.executor, product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory by product: %w", err)
	}

	images, err := u.productImgRepo.ListByProductID(ctx, u.executor, product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load images by product: %w", err)
	}

	result := ProductDetailResult{
		Product:         *product,
		ShopInventories: inventories,
	}

	if len(images) > 0 {
		result.Images = make([]ImageProductDetail, 0, len(images))

		for _, img := range images {
			imageDetail := ImageProductDetail{}

			if variant, ok := img.Variants[domain.ResolutionThumbnail]; ok {
				imageDetail.Thumbnail = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[domain.ResolutionPreview]; ok {
				imageDetail.Preview = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[domain.ResolutionDetail]; ok {
				imageDetail.Detail = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			result.Images = append(result.Images, imageDetail)
		}
	}

	if len(inventories) == 0 {
		return &result, nil
	}

	var shopIDs []uuid.UUID
	for _, inv := range inventories {
		shopIDs = append(shopIDs, inv.ShopID)
	}

	shops, err := u.shopRepo.FindByIDs(ctx, u.executor, shopIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load shops: %w", err)
	}

	shopsMap := make(map[uuid.UUID]shopDomain.Shop)
	for _, s := range shops {
		shopsMap[s.ID] = s
	}

	var (
		totalStock          = 0
		reservedStock       = 0
		operableInventories []inventoryDomain.Inventory
		availability        []ShopAvailabilityResult
	)

	for _, inventory := range inventories {
		shop, ok := shopsMap[inventory.ShopID]
		if !ok || !shop.IsOperable() {
			continue
		}

		totalStock += inventory.TotalStock
		reservedStock += inventory.ReservedStock
		operableInventories = append(operableInventories, inventory)

		availability = append(availability, ShopAvailabilityResult{
			ShopID:   shop.ID,
			ShopName: shop.Name,
			ShopSlug: shop.Slug,
			Stock:    inventory.TotalStock,
		})
	}

	result.ShopInventories = operableInventories
	result.Inventory.TotalStock = totalStock
	result.Inventory.ReservedStock = reservedStock
	result.Availability = availability

	return &result, nil
}
