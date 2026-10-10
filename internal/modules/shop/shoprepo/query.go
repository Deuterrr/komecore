package shoprepo

import (
	"komecore/internal/pagination"

	"github.com/google/uuid"
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
