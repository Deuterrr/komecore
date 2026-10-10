package productusecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type GetProductStatsUsecase struct {
	perfRepo       productrepo.ProductPerformanceRepository
	productImgRepo productrepo.ProductImageRepository
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewGetProductStatsUsecase(
	perfRepo productrepo.ProductPerformanceRepository,
	productImgRepo productrepo.ProductImageRepository,
	fileStore storage.Provider,
	executor transaction.Executor,
) *GetProductStatsUsecase {
	return &GetProductStatsUsecase{
		perfRepo:       perfRepo,
		productImgRepo: productImgRepo,
		fileStore:      fileStore,
		executor:       executor,
	}
}

type GetProductStatsInput struct {
	Page  int
	Limit int
	ID    *string
	Name  *string
	Sort  string
}

func (u *GetProductStatsUsecase) Execute(
	ctx context.Context,
	input GetProductStatsInput,
) ([]productdomain.ProductStats, int, error) {
	var statsSortKeys = map[string]pagination.SortKey{
		"latest":       productrepo.ProductSortLatest,
		"date":         productrepo.ProductSortLatest,
		"name":         productrepo.ProductSortName,
		"price":        productrepo.ProductSortPrice,
		"view_count":   productrepo.ProductSortViewCount,
		"sales_30d":    productrepo.ProductSortSales30d,
		"sales_7d":     productrepo.ProductSortSales7d,
		"revenue":      productrepo.ProductSortRevenue,
		"gross_margin": productrepo.ProductSortGrossMargin,
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
				By:        productrepo.ProductSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	params := productrepo.GetProductStatsParams{
		ID:   input.ID,
		Name: input.Name,
		Pagination: pagination.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	stats, total, err := u.perfRepo.GetProductStats(ctx, u.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load product stats: %w", err)
	}
	if len(stats) == 0 {
		return []productdomain.ProductStats{}, total, nil
	}

	productIDs := make([]uuid.UUID, 0, len(stats))
	for _, stat := range stats {
		productIDs = append(productIDs, stat.Product.ID)
	}

	imagesMap, err := u.productImgRepo.ListByProductIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load images for product stats: %w", err)
	}

	for i := range stats {
		images := imagesMap[stats[i].Product.ID]
		if len(images) > 0 {
			key := images[0].Variants[productdomain.ResolutionThumbnail].Key
			stats[i].Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}
	}

	return stats, total, nil
}
