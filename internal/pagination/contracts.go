package pagination

type (
	SortKey       string
	SortDirection string
)

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type Sort struct {
	By        SortKey
	Direction SortDirection
}

type Sorts []Sort

type Pagination struct {
	Page  int
	Limit int
}

// PaginationMeta represents standard pagination envelope metadata.
type PaginationMeta struct {
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalItems int64   `json:"total_items"`
	TotalPages int     `json:"total_pages"`
	NextCursor *string `json:"next_cursor"`
}

// CalculateTotalPages calculates ceiling(totalItems / limit).
func CalculateTotalPages(totalItems int64, limit int) int {
	if limit <= 0 || totalItems <= 0 {
		return 0
	}
	return int((totalItems + int64(limit) - 1) / int64(limit))
}

// NewPaginationMeta constructs a normalized PaginationMeta instance with next_cursor set to nil.
func NewPaginationMeta(page, limit int, totalItems int64) PaginationMeta {
	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: CalculateTotalPages(totalItems, limit),
		NextCursor: nil,
	}
}
