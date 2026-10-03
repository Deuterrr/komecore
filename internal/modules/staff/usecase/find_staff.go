package usecase

import (
	"context"
	"fmt"
	"strings"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/domain"
	"komecore/internal/modules/staff/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type FindStaffUsecase struct {
	executor  transaction.Executor
	staffRepo repository.StaffRepository
}

func NewFindStaffUsecase(
	executor transaction.Executor,
	staffRepo repository.StaffRepository,
) *FindStaffUsecase {
	return &FindStaffUsecase{
		executor:  executor,
		staffRepo: staffRepo,
	}
}

type FindStaffInput struct {
	Page  int
	Limit int
	ID    *uuid.UUID
	Name  *string
	Sort  string
}

func (u *FindStaffUsecase) Execute(
	ctx context.Context,
	input FindStaffInput,
) ([]domain.StaffProfile, int, error) {
	var staffSortKeys = map[string]query.SortKey{
		"latest":   repository.StaffSortLatest,
		"modified": repository.StaffSortModify,
		// "name":     repository.StaffSortName,
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

			sortKey, exists := staffSortKeys[key]
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
				By:        repository.StaffSortLatest,
				Direction: query.SortDesc,
			},
		}
	}

	params := repository.FindStaffParams{
		ID: input.ID,
		Pagination: query.Pagination{
			Page:  input.Page,
			Limit: input.Limit,
		},
		Sorts: sorts,
	}

	staff, total, err := u.staffRepo.FindStaff(ctx, u.executor, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load staff: %w", err)
	}
	if len(staff) == 0 {
		return nil, 0, apperrors.NewNotFound("staff not available at the moment")
	}

	return staff, total, nil
}
