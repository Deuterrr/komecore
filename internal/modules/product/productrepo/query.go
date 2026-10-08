package productrepo

import (
	"io"

	"komecore/internal/modules/product/productdomain"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

var (
	ProductSortLatest   query.SortKey = "latest"
	ProductSortName     query.SortKey = "name"
	ProductSortPrice    query.SortKey = "price"
	ProductSortWeight   query.SortKey = "weight"
	ProductSortStatus   query.SortKey = "status"
	ProductSortModified query.SortKey = "modified"
	ProductSortArchived query.SortKey = "archived"
	ProductSortStock    query.SortKey = "stock"

	ProductSortViewCount   query.SortKey = "view_count"
	ProductSortSales30d    query.SortKey = "sales_velocity_30d"
	ProductSortSales7d     query.SortKey = "sales_velocity_7d"
	ProductSortRevenue     query.SortKey = "revenue_contribution"
	ProductSortGrossMargin query.SortKey = "gross_margin_pct"
)

type GetProductStatsParams struct {
	ID   *string
	Name *string

	query.Pagination
	query.Sorts
}

type FindProductParams struct {
	ID              *string
	Name            *string
	ShopID          *uuid.UUID
	ShopSlug        *string
	Status          *string
	ExcludeArchived bool

	query.Pagination
	query.Sorts
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
