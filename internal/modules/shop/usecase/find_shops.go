package usecase

import (
	"context"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type FindShopsUsecase struct {
	executor transaction.Executor
	shopRepo repository.ShopRepository
}

func NewFindShopsUsecase(
	executor transaction.Executor,
	shopRepo repository.ShopRepository,
) *FindShopsUsecase {
	return &FindShopsUsecase{
		executor: executor,
		shopRepo: shopRepo,
	}
}

type FindShopsInput struct {
	Page           int
	Limit          int
	ID             *string
	ShopIDs        []uuid.UUID
	Name           *string
	Slug           *string
	IsActive       *bool
	ApprovalStatus *string
	Sort           string
}

func (u *FindShopsUsecase) Execute(
	ctx context.Context,
	input FindShopsInput,
) ([]domain.Shop, int, error) {
	var shopSortKeys = map[string]query.SortKey{
		"name":     repository.ShopSortName,
		"active":   repository.ShopSortActive,
		"date":     repository.ShopSortLatest,
		"modified": repository.ShopSortModify,
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

			sortKey, exists := shopSortKeys[key]
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
				By:        repository.ShopSortLatest,
				Direction: query.SortDesc,
			},
		}
	}

	approvalStatus := input.ApprovalStatus
	if approvalStatus == nil && input.IsActive != nil && *input.IsActive {
		approvedStr := string(domain.ShopApprovalStatusApproved)
		approvalStatus = &approvedStr
	}

	params := repository.FindShopsParams{
		ID:             input.ID,
		ShopIDs:        input.ShopIDs,
		Name:           input.Name,
		Slug:           input.Slug,
		IsActive:       input.IsActive,
		ApprovalStatus: approvalStatus,
		Pagination: query.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	shops, total, err := u.shopRepo.FindByParams(ctx, u.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load shops: %w", err)
	}
	if len(shops) == 0 {
		return []domain.Shop{}, 0, nil
	}

	return shops, total, nil
}
