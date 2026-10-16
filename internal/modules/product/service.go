package product

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/cache"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/shop"
	"komecore/internal/pagination"
	appclock "komecore/pkg/clock"
	image "komecore/pkg/imageutil"
	"komecore/pkg/slug"

	"github.com/google/uuid"
)

var specs = []image.VariantSpec{
	{Type: ResolutionThumbnail, Width: 150},
	{Type: ResolutionPreview, Width: 600},
	{Type: ResolutionDetail, Width: 1200},
}

type Service struct {
	executor       transaction.Executor
	transactor     transaction.Transactor
	productRepo    Repository
	inventoryRepo  inventory.Repository
	productImgRepo ImageRepository
	perfRepo       PerformanceRepository
	shopRepo       shop.Repository
	slugGen        slug.Generator
	resolutionGen  image.VariantCreator
	fileStore      storage.Provider
	cache          cache.Cache
}

func NewService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	productRepo Repository,
	inventoryRepo inventory.Repository,
	productImgRepo ImageRepository,
	perfRepo PerformanceRepository,
	shopRepo shop.Repository,
	slugGen slug.Generator,
	resolutionGen image.VariantCreator,
	fileStore storage.Provider,
) *Service {
	return &Service{
		executor:       executor,
		transactor:     transactor,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		productImgRepo: productImgRepo,
		perfRepo:       perfRepo,
		shopRepo:       shopRepo,
		slugGen:        slugGen,
		resolutionGen:  resolutionGen,
		fileStore:      fileStore,
	}
}

func (s *Service) WithCache(c cache.Cache) *Service {
	s.cache = c
	return s
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
	Product   Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventory.Inventory
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

func (s *Service) FindProducts(ctx context.Context, input FindProductsInput) ([]ProductCatalogResult, int, error) {
	var cacheKey string
	if s.cache != nil {
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
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached.Results, cached.Total, nil
		}
	}

	var productSortKeys = map[string]pagination.SortKey{
		"latest":    ProductSortLatest,
		"date":      ProductSortLatest,
		"name":      ProductSortName,
		"price":     ProductSortPrice,
		"weight":    ProductSortWeight,
		"status":    ProductSortStatus,
		"modified":  ProductSortModified,
		"archived":  ProductSortArchived,
		"stock":     ProductSortStock,
		"relevance": ProductSortRelevance,
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
					By:        ProductSortRelevance,
					Direction: pagination.SortDesc,
				},
			}
		} else {
			sorts = pagination.Sorts{
				{
					By:        ProductSortLatest,
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

	params := FindProductParams{
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

	products, total, err := s.productRepo.FindProductsWithInventory(ctx, s.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load products: %w", err)
	}
	if len(products) == 0 {
		return []ProductCatalogResult{}, total, nil
	}

	productIDs := make([]uuid.UUID, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.Product.ID)
	}

	// Shop data is derived from product inventory records
	// Products are not queried directly by shop;
	// instead shops are inferred via inventory ownership
	inventoryMap, err := s.inventoryRepo.ListByProductIDs(ctx, s.executor, productIDs)
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

	shops, err := s.shopRepo.FindByIDs(ctx, s.executor, shopIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load shops for inventories: %w", err)
	}

	shopsMap := make(map[uuid.UUID]shop.Shop)
	for _, sh := range shops {
		shopsMap[sh.ID] = sh
	}

	imagesMap, err := s.productImgRepo.ListByProductIDs(ctx, s.executor, productIDs)
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

		for _, inv := range inventories {
			if sh, ok := shopsMap[inv.ShopID]; ok {
				if !sh.IsOperable() {
					continue
				}
				totalStock += inv.TotalStock
				reservedStock += inv.ReservedStock

				availability = append(availability, ShopAvailabilityResult{
					ShopID:   sh.ID,
					ShopName: sh.Name,
					ShopSlug: sh.Slug,
					Stock:    inv.TotalStock,
				})
			}
		}

		result.Inventory.TotalStock = totalStock
		result.Inventory.ReservedStock = reservedStock
		result.Availability = availability

		images := imagesMap[p.Product.ID]
		if len(images) > 0 {
			key := images[0].Variants[ResolutionThumbnail].Key
			result.Images.Thumbnail = s.fileStore.PublicURL(key, "public-assets")
		}

		results = append(results, result)
	}

	if s.cache != nil && cacheKey != "" {
		_ = s.cache.Set(ctx, cacheKey, findProductsCacheEntry{Results: results, Total: total}, 5*time.Minute)
	}

	return results, total, nil
}

type ImageProductDetail struct {
	Thumbnail string
	Detail    string
	Preview   string
}

type ProductDetailResult struct {
	Product   Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventory.Inventory
	Images          []ImageProductDetail
	Availability    []ShopAvailabilityResult
}

func (s *Service) GetProduct(
	ctx context.Context,
	slug string,
) (*ProductDetailResult, error) {
	var cacheKey string
	if s.cache != nil {
		cacheKey = fmt.Sprintf("cache:product:slug:%s", slug)
		var cached ProductDetailResult
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	prod, err := s.productRepo.GetBySlug(ctx, s.executor, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to load products with inventory: %w", err)
	}
	if prod == nil {
		return nil, nil
	}

	if s.perfRepo != nil {
		go func() {
			_ = s.perfRepo.IncrementViewCount(context.Background(), s.executor, prod.ID)
		}()
	}

	var inventories []inventory.Inventory
	if s.inventoryRepo != nil {
		var err error
		inventories, err = s.inventoryRepo.ListByProductID(ctx, s.executor, prod.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load inventory by product: %w", err)
		}
	}

	var images []ProductImage
	if s.productImgRepo != nil {
		var err error
		images, err = s.productImgRepo.ListByProductID(ctx, s.executor, prod.ID)
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

			if variant, ok := img.Variants[ResolutionThumbnail]; ok {
				imageDetail.Thumbnail = s.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[ResolutionPreview]; ok {
				imageDetail.Preview = s.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			if variant, ok := img.Variants[ResolutionDetail]; ok {
				imageDetail.Detail = s.fileStore.PublicURL(
					variant.Key,
					"public-assets",
				)
			}

			result.Images = append(result.Images, imageDetail)
		}
	}

	if len(inventories) == 0 {
		if s.cache != nil && cacheKey != "" {
			_ = s.cache.Set(ctx, cacheKey, result, 15*time.Minute)
		}
		return &result, nil
	}

	var shopIDs []uuid.UUID
	for _, inv := range inventories {
		shopIDs = append(shopIDs, inv.ShopID)
	}

	shops, err := s.shopRepo.FindByIDs(ctx, s.executor, shopIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load shops: %w", err)
	}

	shopsMap := make(map[uuid.UUID]shop.Shop)
	for _, sh := range shops {
		shopsMap[sh.ID] = sh
	}

	var (
		totalStock          = 0
		reservedStock       = 0
		operableInventories []inventory.Inventory
		availability        []ShopAvailabilityResult
	)

	for _, inv := range inventories {
		sh, ok := shopsMap[inv.ShopID]
		if !ok || !sh.IsOperable() {
			continue
		}

		totalStock += inv.TotalStock
		reservedStock += inv.ReservedStock
		operableInventories = append(operableInventories, inv)

		availability = append(availability, ShopAvailabilityResult{
			ShopID:   sh.ID,
			ShopName: sh.Name,
			ShopSlug: sh.Slug,
			Stock:    inv.TotalStock,
		})
	}

	result.ShopInventories = operableInventories
	result.Inventory.TotalStock = totalStock
	result.Inventory.ReservedStock = reservedStock
	result.Availability = availability

	if s.cache != nil && cacheKey != "" {
		_ = s.cache.Set(ctx, cacheKey, result, 15*time.Minute)
	}

	return &result, nil
}

type GetProductStatsInput struct {
	Page  int
	Limit int
	ID    *string
	Name  *string
	Sort  string
}

func (s *Service) GetProductStats(
	ctx context.Context,
	input GetProductStatsInput,
) ([]ProductStats, int, error) {
	var statsSortKeys = map[string]pagination.SortKey{
		"latest":       ProductSortLatest,
		"date":         ProductSortLatest,
		"name":         ProductSortName,
		"price":        ProductSortPrice,
		"view_count":   ProductSortViewCount,
		"sales_30d":    ProductSortSales30d,
		"sales_7d":     ProductSortSales7d,
		"revenue":      ProductSortRevenue,
		"gross_margin": ProductSortGrossMargin,
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

			sortKey, exists := statsSortKeys[key]
			if exists {
				sorts = append(sorts, pagination.Sort{
					By:        sortKey,
					Direction: dir,
				})
			}
		}
	}

	if len(sorts) == 0 {
		sorts = pagination.Sorts{
			{
				By:        ProductSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	params := GetProductStatsParams{
		ID:   input.ID,
		Name: input.Name,
		Pagination: pagination.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	stats, total, err := s.perfRepo.GetProductStats(ctx, s.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load product stats: %w", err)
	}
	if len(stats) == 0 {
		return []ProductStats{}, total, nil
	}

	productIDs := make([]uuid.UUID, 0, len(stats))
	for _, stat := range stats {
		productIDs = append(productIDs, stat.Product.ID)
	}

	imagesMap, err := s.productImgRepo.ListByProductIDs(ctx, s.executor, productIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load images for product stats: %w", err)
	}

	for i := range stats {
		images := imagesMap[stats[i].Product.ID]
		if len(images) > 0 {
			key := images[0].Variants[ResolutionThumbnail].Key
			stats[i].Thumbnail = s.fileStore.PublicURL(key, "public-assets")
		}
	}

	return stats, total, nil
}

type SaveProductInput struct {
	ID                   *uuid.UUID
	SKU                  string
	Name                 string
	Description          *string
	Status               string
	Price                int64
	Weight               *float64
	CostPrice            *int64
	SupplierLeadTimeDays *int
}

func (s *Service) SaveProduct(
	ctx context.Context,
	input SaveProductInput,
) error {
	now := appclock.Now()

	var productID uuid.UUID
	if input.ID == nil {
		productID = uuid.New()
	} else {
		productID = *input.ID
	}

	prodStatus := ProductStatus(input.Status)
	if prodStatus == "" {
		prodStatus = ProductStatusActive
	}

	prod := &Product{
		ID:          productID,
		SKU:         input.SKU,
		Name:        input.Name,
		Slug:        s.slugGen.Generate(input.Name),
		Description: input.Description,
		Status:      prodStatus,
		Price:       input.Price,
		Weight:      input.Weight,
		CreatedAt:   now,
	}
	if err := prod.Validate(); err != nil {
		if errors.Is(err, ErrInvalidProductName) ||
			errors.Is(err, ErrInvalidProductPrice) {
			return apperror.NewInvalidInput(err.Error())
		}

		return err
	}

	perf := ProductPerformance{
		ProductID:            productID,
		CostPrice:            input.CostPrice,
		SupplierLeadTimeDays: input.SupplierLeadTimeDays,
	}

	if err := s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.productRepo.Save(ctx, exec,
			prod,
		); err != nil {
			return fmt.Errorf("failed to save product: %w", err)
		}

		if err := s.perfRepo.UpsertPerformance(ctx, exec,
			perf,
			prod.Price,
		); err != nil {
			return fmt.Errorf("failed to save product performance: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.DeletePattern(ctx, "cache:product*")
	}

	return nil
}

func (s *Service) DeleteProduct(
	ctx context.Context,
	id uuid.UUID,
) error {
	prod, err := s.productRepo.GetByID(ctx, s.executor, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve product: %w", err)
	}
	if prod == nil {
		return apperror.NewNotFound("product not found")
	}

	if err := s.productRepo.Delete(ctx, s.executor, prod.ID); err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.DeletePattern(ctx, "cache:product*")
	}

	return nil
}

type ProductImageInput struct {
	Data         []byte
	OriginalName string
	MIMEType     string
	SizeBytes    int64
	IsPrimary    bool
	DisplayOrder int
}

type AddProductImageInput struct {
	ProductID uuid.UUID
	Images    []ProductImageInput
}

type productToVariants struct {
	index       int
	variantType image.ResolutionType
}

func (s *Service) AddProductImages(
	ctx context.Context,
	input AddProductImageInput,
) error {
	prod, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to get product: %w", err)
	}

	if prod == nil {
		return apperror.NewNotFound("product not found")
	}

	var (
		uploadImages  []storage.UploadInput
		productImages []ProductImage
		mappings      []productToVariants
		now           = appclock.Now()
	)

	for _, img := range input.Images {
		productImage := ProductImage{
			ID:           uuid.New(),
			ProductID:    prod.ID,
			Variants:     make(map[image.ResolutionType]ImageVariant),
			IsPrimary:    img.IsPrimary,
			DisplayOrder: img.DisplayOrder,
			Metadata: ProductImageMetadata{
				OriginalName: img.OriginalName,
				MIMEType:     image.MIME(img.MIMEType),
				SizeBytes:    img.SizeBytes,
			},
			CreatedAt: now,
		}

		productImageIndex := len(productImages)
		productImages = append(productImages, productImage)

		variants, err := s.resolutionGen.GenerateVariants(
			img.Data,
			image.MIME(img.MIMEType),
			specs,
		)
		if err != nil {
			return fmt.Errorf("failed to create variants of %s: %w", img.OriginalName, err)
		}

		for _, variant := range variants {
			key, err := productImage.BuildObjectKey()
			if err != nil {
				return fmt.Errorf("failed to build %s key: %w", variant.Type, err)
			}

			uploadImages = append(uploadImages, storage.UploadInput{
				Bucket:      "public-assets",
				Key:         key,
				File:        bytes.NewReader(variant.Data),
				ContentType: string(variant.MIMEType),
			})

			mappings = append(mappings, productToVariants{
				index:       productImageIndex,
				variantType: variant.Type,
			})
		}
	}

	responses, err := s.fileStore.UploadMany(uploadImages)
	if err != nil {
		return fmt.Errorf("failed to upload images: %w", err)
	}

	for i, resp := range responses {
		m := mappings[i]

		productImages[m.index].Variants[m.variantType] =
			ImageVariant{
				Type: m.variantType,
				Key:  resp.Key,
			}
	}

	return s.transactor.WithinTransaction(
		ctx,
		func(exec transaction.Executor) error {
			if err := s.productImgRepo.Create(ctx, exec, productImages); err != nil {
				return fmt.Errorf("failed to save product image: %w", err)
			}

			return nil
		},
	)
}
