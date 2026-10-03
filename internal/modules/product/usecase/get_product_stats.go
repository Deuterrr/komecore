package usecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type GetProductStatsUsecase struct {
	perfRepo       repository.ProductPerformanceRepository
	productImgRepo repository.ProductImageRepository
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewGetProductStatsUsecase(
	perfRepo repository.ProductPerformanceRepository,
	productImgRepo repository.ProductImageRepository,
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
) ([]domain.ProductStats, int, error) {
	var statsSortKeys = map[string]query.SortKey{
		"latest":       repository.ProductSortLatest,
		"date":         repository.ProductSortLatest,
		"name":         repository.ProductSortName,
		"price":        repository.ProductSortPrice,
		"view_count":   repository.ProductSortViewCount,
		"sales_30d":    repository.ProductSortSales30d,
		"sales_7d":     repository.ProductSortSales7d,
		"revenue":      repository.ProductSortRevenue,
		"gross_margin": repository.ProductSortGrossMargin,
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

			sortKey, exists := statsSortKeys[key]
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

	params := repository.GetProductStatsParams{
		ID:   input.ID,
		Name: input.Name,
		Pagination: query.Pagination{
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
		return []domain.ProductStats{}, total, nil
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
			key := images[0].Variants[domain.ResolutionThumbnail].Key
			stats[i].Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}
	}

	return stats, total, nil
}
