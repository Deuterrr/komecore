package usecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/cache"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
	shopDomain "komecore/internal/modules/shop/domain"
	shopRepo "komecore/internal/modules/shop/repository"
	query "komecore/internal/shared/query"
	"time"

	"github.com/google/uuid"
)

type FindProductsUsecase struct {
	productRepo    repository.ProductRepository
	inventoryRepo  inventoryRepo.InventoryRepository
	productImgRepo repository.ProductImageRepository
	shopRepo       shopRepo.ShopRepository
	fileStore      storage.Provider
	executor       transaction.Executor
	cache          cache.Cache
}

func NewFindProductsUsecase(
	productRepo repository.ProductRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productImgRepo repository.ProductImageRepository,
	shopRepo shopRepo.ShopRepository,
	fileStore storage.Provider,
	executor transaction.Executor,
) *FindProductsUsecase {
	return &FindProductsUsecase{
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		productImgRepo: productImgRepo,
		shopRepo:       shopRepo,
		fileStore:      fileStore,
		executor:       executor,
	}
}

func (u *FindProductsUsecase) WithCache(c cache.Cache) *FindProductsUsecase {
	u.cache = c
	return u
}

type findProductsCacheEntry struct {
	Results []ProductCatalogResult `json:"results"`
	Total   int                    `json:"total"`
}

type ShopAvailabilityResult struct {
	ShopID   uuid.UUID
	ShopName string
	ShopSlug string
	Stock    int
}

type ProductCatalogResult struct {
	Product   domain.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventoryDomain.Inventory
	Images          struct {
		Thumbnail string
	}
	Availability []ShopAvailabilityResult
}

type FindProductsInput struct {
	Page            int
	Limit           int
	ID              *string
	Name            *string
	ShopID          *string
	ShopSlug        *string
	Status          *string
	ExcludeArchived bool
	Sort            string
}

func (u *FindProductsUsecase) Execute(
	ctx context.Context,
	input FindProductsInput,
) ([]ProductCatalogResult, int, error) {
	var cacheKey string
	if u.cache != nil {
		nameStr := ""
		if input.Name != nil {
			nameStr = *input.Name
		}
		statusStr := ""
		if input.Status != nil {
			statusStr = *input.Status
		}
		cacheKey = fmt.Sprintf("cache:products:list:%s:%s:%s:%d:%d", nameStr, statusStr, input.Sort, input.Page, input.Limit)
		var cached findProductsCacheEntry
		if err := u.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached.Results, cached.Total, nil
		}
	}

	var productSortKeys = map[string]query.SortKey{
		"latest":   repository.ProductSortLatest,
		"date":     repository.ProductSortLatest,
		"name":     repository.ProductSortName,
		"price":    repository.ProductSortPrice,
		"weight":   repository.ProductSortWeight,
		"status":   repository.ProductSortStatus,
		"modified": repository.ProductSortModified,
		"archived": repository.ProductSortArchived,
		"stock":    repository.ProductSortStock,
	}

	var sorts query.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir query.SortDirection = query.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = query.SortAsc
				}
			}

			sortKey, exists := productSortKeys[key]
			if exists {
				sorts = append(sorts, query.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = query.Sorts{
			{
				By:        repository.ProductSortLatest,
				Direction: query.SortDesc,
			},
		}
	}

	var shopUUID *uuid.UUID
	if input.ShopID != nil && *input.ShopID != "" {
		if parsed, err := uuid.Parse(*input.ShopID); err == nil {
			shopUUID = &parsed
		}
	}

	params := repository.FindProductParams{
		ID:              input.ID,
		Name:            input.Name,
		ShopID:          shopUUID,
		ShopSlug:        input.ShopSlug,
		Status:          input.Status,
		ExcludeArchived: input.ExcludeArchived,
		Pagination: query.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	products, total, err := u.productRepo.FindProductsWithInventory(ctx, u.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load products: %w", err)
	}
	if len(products) == 0 {
		return []ProductCatalogResult{}, total, nil
	}

	productIDs := make([]uuid.UUID, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.Product.ID)
	}

	// Shop data is derived from product inventory records
	// Products are not queried directly by shop;
	// instead shops are inferred via inventory ownership
	inventoryMap, err := u.inventoryRepo.ListByProductIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load inventory for products: %w", err)
	}

	// Deduplicate shop IDs from product inventories
	shopIDsMap := make(map[uuid.UUID]bool)
	for _, inventories := range inventoryMap {
		for _, inv := range inventories {
			shopIDsMap[inv.ShopID] = true
		}
	}

	shopIDs := make([]uuid.UUID, 0, len(shopIDsMap))
	for id := range shopIDsMap {
		shopIDs = append(shopIDs, id)
	}

	shops, err := u.shopRepo.FindByIDs(ctx, u.executor, shopIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load shops for inventories: %w", err)
	}

	shopsMap := make(map[uuid.UUID]shopDomain.Shop)
	for _, s := range shops {
		shopsMap[s.ID] = s
	}

	imagesMap, err := u.productImgRepo.ListByProductIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load images for products: %w", err)
	}

	results := make([]ProductCatalogResult, 0, len(products))
	for _, p := range products {
		inventories := inventoryMap[p.Product.ID]

		result := ProductCatalogResult{
			Product:         p.Product,
			ShopInventories: inventories,
		}

		var (
			totalStock    = 0
			reservedStock = 0
			availability  []ShopAvailabilityResult
		)

		for _, inventory := range inventories {
			if shop, ok := shopsMap[inventory.ShopID]; ok {
				if !shop.IsOperable() {
					continue
				}
				totalStock += inventory.TotalStock
				reservedStock += inventory.ReservedStock

				availability = append(availability, ShopAvailabilityResult{
					ShopID:   shop.ID,
					ShopName: shop.Name,
					ShopSlug: shop.Slug,
					Stock:    inventory.TotalStock,
				})
			}
		}

		result.Inventory.TotalStock = totalStock
		result.Inventory.ReservedStock = reservedStock
		result.Availability = availability

		images := imagesMap[p.Product.ID]
		if len(images) > 0 {
			key := images[0].Variants[domain.ResolutionThumbnail].Key
			result.Images.Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}

		results = append(results, result)
	}

	if u.cache != nil && cacheKey != "" {
		_ = u.cache.Set(ctx, cacheKey, findProductsCacheEntry{Results: results, Total: total}, 5*time.Minute)
	}

	return results, total, nil
}
