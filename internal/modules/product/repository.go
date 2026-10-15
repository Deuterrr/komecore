package product

import (
	"context"
	"io"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
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
	ProductImageMetadata
}

type UploadedProductImage struct {
	Sequence int

	CatalogURL string
	CartURL    string
	DetailURL  string

	IsPrimary bool
}

type Repository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Product, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		string string,
	) (*Product, error)

	FindProducts(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]Product, int, error)

	FindProductsWithInventory(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]ProductWithInventory, int, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]Product, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		product *Product,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	UpdateRating(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
		averageRating float64,
		reviewCount int,
	) error
}

type ImageRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		images []ProductImage,
	) error

	ListByProductIDs(
		ctx context.Context,
		exec transaction.Executor,
		productIDs []uuid.UUID,
	) (map[uuid.UUID][]ProductImage, error)

	ListByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) ([]ProductImage, error)

	SoftDeleteByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) error
}

type ImageUploadService interface {
	Upload(params UploadProductImagesParams) ([]UploadedProductImage, error)
	Delete(assetKeys []string) error
}

type PerformanceRepository interface {
	UpsertPerformance(
		ctx context.Context,
		exec transaction.Executor,
		perf ProductPerformance,
		basePrice int64,
	) error

	IncrementViewCount(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) error

	GetProductStats(
		ctx context.Context,
		exec transaction.Executor,
		params GetProductStatsParams,
	) ([]ProductStats, int, error)
}

type StockHistoryRepository interface {
	RecordStockEvent(
		ctx context.Context,
		exec transaction.Executor,
		event ProductStockEvent,
	) error
}

type ProductRepository = Repository
type ProductImageRepository = ImageRepository
type ProductImageUploadService = ImageUploadService
type ProductPerformanceRepository = PerformanceRepository
type ProductStockHistoryRepository = StockHistoryRepository
