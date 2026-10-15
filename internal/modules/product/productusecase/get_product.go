package productusecase

import (
	"context"
	"fmt"

	"komecore/internal/infra/cache"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"
	"time"

	"github.com/google/uuid"
)

type GetProductUsecase struct {
	executor       transaction.Executor
	fileStore      storage.Provider
	productRepo    product.Repository
	inventoryRepo  inventory.Repository
	productImgRepo product.ImageRepository
	shopRepo       shop.Repository
	perfRepo       product.PerformanceRepository
	cache          cache.Cache
}

func NewGetProductUsecase(
	executor transaction.Executor,
	fileStore storage.Provider,
	productRepo product.Repository,
	inventoryRepo inventory.Repository,
	productImgRepo product.ImageRepository,
	shopRepo shop.Repository,
	perfRepo product.PerformanceRepository,
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

func (u *GetProductUsecase) WithCache(c cache.Cache) *GetProductUsecase {
	u.cache = c
	return u
}

type ImageProductDetail struct {
	Thumbnail string
	Detail    string
	Preview   string
}

type ProductDetailResult struct {
	Product   product.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventory.Inventory
	Images          []ImageProductDetail
	Availability    []ShopAvailabilityResult
}

func (u *GetProductUsecase) Execute(
	ctx context.Context,
	slug string,
) (*ProductDetailResult, error) {
	var cacheKey string
	if u.cache != nil {
		cacheKey = fmt.Sprintf("cache:product:slug:%s", slug)
		var cached ProductDetailResult
		if err := u.cache.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	prod, err := u.productRepo.GetBySlug(ctx, u.executor, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to load products with inventory: %w", err)
	}
	if prod == nil {
		return nil, nil
	}

	if u.perfRepo != nil {
		go func() {
			_ = u.perfRepo.IncrementViewCount(context.Background(), u.executor, prod.ID)
		}()
	}

	var inventories []inventory.Inventory
	if u.inventoryRepo != nil {
		var err error
		inventories, err = u.inventoryRepo.ListByProductID(ctx, u.executor, prod.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load inventory by product: %w", err)
		}
	}

	var images []product.ProductImage
	if u.productImgRepo != nil {
		var err error
		images, err = u.productImgRepo.ListByProductID(ctx, u.executor, prod.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load images by product: %w", err)
		}
	}

	result := ProductDetailResult{
		Product:         *prod,
		ShopInventories: inventories,
	}

	if len(images) > 0 {
		result.Images = make([]ImageProductDetail, 0, len(images))

		for _, img := range images {
			imageDetail := ImageProductDetail{}

			if variant, ok := img.Variants[product.ResolutionThumbnail]; ok {
				imageDetail.Thumbnail = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[product.ResolutionPreview]; ok {
				imageDetail.Preview = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[product.ResolutionDetail]; ok {
				imageDetail.Detail = u.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			result.Images = append(result.Images, imageDetail)
		}
	}

	if len(inventories) == 0 {
		if u.cache != nil && cacheKey != "" {
			_ = u.cache.Set(ctx, cacheKey, result, 15*time.Minute)
		}
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

	shopsMap := make(map[uuid.UUID]shop.Shop)
	for _, s := range shops {
		shopsMap[s.ID] = s
	}

	var (
		totalStock          = 0
		reservedStock       = 0
		operableInventories []inventory.Inventory
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

	if u.cache != nil && cacheKey != "" {
		_ = u.cache.Set(ctx, cacheKey, result, 15*time.Minute)
	}

	return &result, nil
}
