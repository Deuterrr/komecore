package product

import (
	"context"
	"io"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	// Orders products by descending creation timestamp (default storefront order).
	ProductSortLatest pagination.SortKey = "latest"
	// Orders products alphabetically by name.
	ProductSortName pagination.SortKey = "name"
	// Orders products by unit retail price.
	ProductSortPrice pagination.SortKey = "price"
	// Orders products by package shipping weight in grams.
	ProductSortWeight pagination.SortKey = "weight"
	// Orders products by catalog publication status.
	ProductSortStatus pagination.SortKey = "status"
	// Orders products by last modification timestamp.
	ProductSortModified pagination.SortKey = "modified"
	// Orders products by archive timestamp.
	ProductSortArchived pagination.SortKey = "archived"
	// Orders products by available warehouse stock balance.
	ProductSortStock pagination.SortKey = "stock"

	// Orders analytics by total product detail page impressions.
	ProductSortViewCount pagination.SortKey = "view_count"
	// Orders analytics by 30-day trailing sales velocity.
	ProductSortSales30d pagination.SortKey = "sales_velocity_30d"
	// Orders analytics by 7-day short-term sales velocity.
	ProductSortSales7d pagination.SortKey = "sales_velocity_7d"
	// Orders analytics by gross revenue generated.
	ProductSortRevenue pagination.SortKey = "revenue_contribution"
	// Orders analytics by gross profit margin percentage.
	ProductSortGrossMargin pagination.SortKey = "gross_margin_pct"
	// Orders search results by full-text relevance ranking.
	ProductSortRelevance pagination.SortKey = "relevance"
)

// GetProductStatsParams encapsulates filtering and pagination parameters for product performance analytics.
type GetProductStatsParams struct {
	ID   *string
	Name *string

	pagination.Pagination
	pagination.Sorts
}

// FindProductParams encapsulates catalog search filters, shop tenancy, status, and sorting options.
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

// UploadProductImagesParams holds payload and file stream references for product gallery image ingestion.
type UploadProductImagesParams struct {
	ProductID uuid.UUID
	Files     []ImageFile
}

// ImageFile wraps an image stream and its display metadata.
type ImageFile struct {
	File io.Reader
	ProductImageMetadata
}

// UploadedProductImage represents processed CDN asset URLs for responsive storefront rendering.
type UploadedProductImage struct {
	Sequence int

	CatalogURL string
	CartURL    string
	DetailURL  string

	IsPrimary bool
}

// Repository defines persistence operations for merchant catalog products, pricing, and ratings.
type Repository interface {
	// Resolves a product record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Product, error)
	// Looks up an active product by unique URL slug for storefront product detail pages.
	GetBySlug(ctx context.Context, exec transaction.Executor, string string) (*Product, error)

	// Queries paginated products with multi-attribute filtering (merchant shop, search query, status).
	FindProducts(ctx context.Context, exec transaction.Executor, params FindProductParams) ([]Product, int, error)
	// Queries paginated products joined with live warehouse inventory quantities.
	FindProductsWithInventory(ctx context.Context, exec transaction.Executor, params FindProductParams) ([]ProductWithInventory, int, error)
	// Batch-loads product records across multiple IDs to prevent N+1 queries during checkout pricing and cart evaluation.
	FindByIDs(ctx context.Context, exec transaction.Executor, IDs []uuid.UUID) ([]Product, error)

	// Inserts a new catalog item or updates title, description, dimensions, weight, and base price.
	Save(ctx context.Context, exec transaction.Executor, product *Product) error

	// Soft-deletes a product, disabling it from storefront listings while preserving relational order history.
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error

	// Atomically updates aggregated average star rating and review count following customer feedback submission.
	UpdateRating(ctx context.Context, exec transaction.Executor, productID uuid.UUID, averageRating float64, reviewCount int) error
}

// ImageRepository defines persistence operations for catalog product gallery images, thumbnail URLs, and sequence rankings.
type ImageRepository interface {
	// Batch-persists image URLs, CDN keys, and sequence numbers for a product.
	Create(ctx context.Context, exec transaction.Executor, images []ProductImage) error

	// Batch-loads gallery images grouped by product ID to prevent N+1 queries in catalog listing views.
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]ProductImage, error)
	// Queries all gallery images for a single product ordered by display sequence.
	ListByProductID(ctx context.Context, exec transaction.Executor, productID uuid.UUID) ([]ProductImage, error)

	// Soft-deletes all gallery image references associated with a catalog product.
	SoftDeleteByProductID(ctx context.Context, exec transaction.Executor, productID uuid.UUID) error
}

// ImageUploadService coordinates external object storage uploads, thumbnail transformations, and asset deletions.
type ImageUploadService interface {
	Upload(params UploadProductImagesParams) ([]UploadedProductImage, error)
	Delete(assetKeys []string) error
}

// PerformanceRepository defines persistence operations for product telemetry, page views, and sales velocity metrics.
type PerformanceRepository interface {
	// Updates sales velocity, revenue contribution, and margin statistics for a product.
	UpsertPerformance(ctx context.Context, exec transaction.Executor, perf ProductPerformance, basePrice int64) error

	// Atomically increments product page view counter.
	IncrementViewCount(ctx context.Context, exec transaction.Executor, productID uuid.UUID) error

	// Queries aggregated performance metrics for merchant analytics dashboards.
	GetProductStats(ctx context.Context, exec transaction.Executor, params GetProductStatsParams) ([]ProductStats, int, error)
}

// StockHistoryRepository defines append-only persistence for product stock movements, adjustments, reservations, and restocks.
type StockHistoryRepository interface {
	// Appends an immutable stock audit event for inventory reconciliation and shrinkage auditing.
	RecordStockEvent(ctx context.Context, exec transaction.Executor, event ProductStockEvent) error
}
