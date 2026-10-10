package staffrepo

import (
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

var (
	StaffSortLatest pagination.SortKey = "latest"
	// StaffSortName   pagination.SortKey = "name"
	StaffSortModify pagination.SortKey = "modified"
)

type FindStaffParams struct {
	ID *uuid.UUID
	// Name *string

	pagination.Pagination
	pagination.Sorts
}
