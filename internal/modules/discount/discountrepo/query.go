package discountrepo

import "komecore/internal/pagination"

type ListCouponsParams struct {
	Code     *string
	IsActive *bool

	pagination.Pagination
	pagination.Sorts
}
