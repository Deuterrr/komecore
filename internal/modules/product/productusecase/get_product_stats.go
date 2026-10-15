package productusecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type GetProductStatsUsecase struct {
	perfRepo       product.PerformanceRepository
	productImgRepo product.ImageRepository
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewGetProductStatsUsecase(
	perfRepo product.PerformanceRepository,
	productImgRepo product.ImageRepository,
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
) ([]product.ProductStats, int, error) {
	var statsSortKeys = map[string]pagination.SortKey{
		"latest":       product.ProductSortLatest,
		"date":         product.ProductSortLatest,
		"name":         product.ProductSortName,
		"price":        product.ProductSortPrice,
		"view_count":   product.ProductSortViewCount,
		"sales_30d":    product.ProductSortSales30d,
		"sales_7d":     product.ProductSortSales7d,
		"revenue":      product.ProductSortRevenue,
		"gross_margin": product.ProductSortGrossMargin,
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
				By:        product.ProductSortLatest,
				Direction: pagination.SortDesc,
			},
		}
	}

	params := product.GetProductStatsParams{
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
		return []product.ProductStats{}, total, nil
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
			key := images[0].Variants[product.ResolutionThumbnail].Key
			stats[i].Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}
	}

	return stats, total, nil
}
