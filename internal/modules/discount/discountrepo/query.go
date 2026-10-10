package discountrepo

import query "komecore/internal/shared/query"

type ListCouponsParams struct {
	Code     *string
	IsActive *bool

	query.Pagination
	query.Sorts
}
