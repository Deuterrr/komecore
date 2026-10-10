package productusecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/cache"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/modules/shop/shopdomain"
	"komecore/internal/modules/shop/shoprepo"
	"komecore/internal/pagination"
	"time"

	"github.com/google/uuid"
)

type FindProductsUsecase struct {
	productRepo    productrepo.ProductRepository
	inventoryRepo  inventoryrepo.InventoryRepository
	productImgRepo productrepo.ProductImageRepository
	shopRepo       shoprepo.ShopRepository
	fileStore      storage.Provider
	executor       transaction.Executor
	cache          cache.Cache
}

func NewFindProductsUsecase(
	productRepo productrepo.ProductRepository,
	inventoryRepo inventoryrepo.InventoryRepository,
	productImgRepo productrepo.ProductImageRepository,
	shopRepo shoprepo.ShopRepository,
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
	Product   productdomain.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventorydomain.Inventory
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
	SearchQuery     *string
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
		queryStr := ""
		if input.SearchQuery != nil {
			queryStr = *input.SearchQuery
		}
		statusStr := ""
		if input.Status != nil {
			statusStr = *input.Status
		}
		cacheKey = fmt.Sprintf("cache:products:list:%s:%s:%s:%s:%d:%d", nameStr, queryStr, statusStr, input.Sort, input.Page, input.Limit)
		var cached findProductsCacheEntry
		if err := u.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached.Results, cached.Total, nil
		}
	}

	var productSortKeys = map[string]pagination.SortKey{
		"latest":    productrepo.ProductSortLatest,
		"date":      productrepo.ProductSortLatest,
		"name":      productrepo.ProductSortName,
		"price":     productrepo.ProductSortPrice,
		"weight":    productrepo.ProductSortWeight,
		"status":    productrepo.ProductSortStatus,
		"modified":  productrepo.ProductSortModified,
		"archived":  productrepo.ProductSortArchived,
		"stock":     productrepo.ProductSortStock,
		"relevance": productrepo.ProductSortRelevance,
	}

	var sorts pagination.Sorts
	if input.Sort != "" {
		parts := strings.SplitSeq(input.Sort, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ":")
			key := strings.TrimSpace(subparts[0])

			var dir pagination.SortDirection = pagination.SortDesc
			if len(subparts) > 1 {
				d := strings.ToLower(strings.TrimSpace(subparts[1]))
				if d == "asc" {
					dir = pagination.SortAsc
				}
			}

			sortKey, exists := productSortKeys[key]
			if exists {
				sorts = append(sorts, pagination.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		if input.SearchQuery != nil && strings.TrimSpace(*input.SearchQuery) != "" {
			sorts = pagination.Sorts{
				{
					By:        productrepo.ProductSortRelevance,
					Direction: pagination.SortDesc,
				},
			}
		} else {
			sorts = pagination.Sorts{
				{
					By:        productrepo.ProductSortLatest,
					Direction: pagination.SortDesc,
				},
			}
		}
	}

	var shopUUID *uuid.UUID
	if input.ShopID != nil && *input.ShopID != "" {
		if parsed, err := uuid.Parse(*input.ShopID); err == nil {
			shopUUID = &parsed
		}
	}

	params := productrepo.FindProductParams{
		ID:              input.ID,
		Name:            input.Name,
		SearchQuery:     input.SearchQuery,
		ShopID:          shopUUID,
		ShopSlug:        input.ShopSlug,
		Status:          input.Status,
		ExcludeArchived: input.ExcludeArchived,
		Pagination: pagination.Pagination{
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

	shopsMap := make(map[uuid.UUID]shopdomain.Shop)
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
			key := images[0].Variants[productdomain.ResolutionThumbnail].Key
			result.Images.Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}

		results = append(results, result)
	}

	if u.cache != nil && cacheKey != "" {
		_ = u.cache.Set(ctx, cacheKey, findProductsCacheEntry{Results: results, Total: total}, 5*time.Minute)
	}

	return results, total, nil
}
