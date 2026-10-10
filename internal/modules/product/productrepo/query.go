package productrepo

import (
	"io"

	"komecore/internal/modules/product/productdomain"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

var (
	ProductSortLatest   pagination.SortKey = "latest"
	ProductSortName     pagination.SortKey = "name"
	ProductSortPrice    pagination.SortKey = "price"
	ProductSortWeight   pagination.SortKey = "weight"
	ProductSortStatus   pagination.SortKey = "status"
	ProductSortModified pagination.SortKey = "modified"
	ProductSortArchived pagination.SortKey = "archived"
	ProductSortStock    pagination.SortKey = "stock"

	ProductSortViewCount   pagination.SortKey = "view_count"
	ProductSortSales30d    pagination.SortKey = "sales_velocity_30d"
	ProductSortSales7d     pagination.SortKey = "sales_velocity_7d"
	ProductSortRevenue     pagination.SortKey = "revenue_contribution"
	ProductSortGrossMargin pagination.SortKey = "gross_margin_pct"
	ProductSortRelevance   pagination.SortKey = "relevance"
)

type GetProductStatsParams struct {
	ID   *string
	Name *string

	pagination.Pagination
	pagination.Sorts
}

type FindProductParams struct {
	ID              *string
	Name            *string
	SearchQuery     *string
	ShopID          *uuid.UUID
	ShopSlug        *string
	Status          *string
	ExcludeArchived bool

	pagination.Pagination
	pagination.Sorts
}

type UploadProductImagesParams struct {
	ProductID uuid.UUID
	Files     []ImageFile
}

type ImageFile struct {
	File io.Reader
	productdomain.ProductImageMetadata
}

type UploadedProductImage struct {
	Sequence int

	CatalogURL string
	CartURL    string
	DetailURL  string

	IsPrimary bool
}
