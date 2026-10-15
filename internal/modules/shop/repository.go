package shop

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	ShopSortLatest pagination.SortKey = "latest"
	ShopSortName   pagination.SortKey = "name"
	ShopSortActive pagination.SortKey = "active"
	ShopSortModify pagination.SortKey = "modify"
)

type FindShopsParams struct {
	ID             *string
	ShopIDs        []uuid.UUID
	Name           *string
	Slug           *string
	IsActive       *bool
	ApprovalStatus *string

	Pagination pagination.Pagination
	Sorts      pagination.Sorts
}

type Repository interface {
	FindByParams(
		ctx context.Context,
		exec transaction.Executor,
		params FindShopsParams,
	) ([]Shop, int, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Shop, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		slug string,
	) (*Shop, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]Shop, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		shop Shop,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}

type ShopRepository = Repository
