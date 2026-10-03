package query

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
