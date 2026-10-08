package staffrepo

import (
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

var (
	StaffSortLatest query.SortKey = "latest"
	// StaffSortName   query.SortKey = "name"
	StaffSortModify query.SortKey = "modified"
)

type FindStaffParams struct {
	ID *uuid.UUID
	// Name *string

	query.Pagination
	query.Sorts
}
