package productrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"

	"github.com/google/uuid"
)

type ProductRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*productdomain.Product, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		string string,
	) (*productdomain.Product, error)

	FindProducts(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]productdomain.Product, int, error)

	FindProductsWithInventory(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]productdomain.ProductWithInventory, int, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]productdomain.Product, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		product *productdomain.Product,
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

type ProductImageRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		images []productdomain.ProductImage,
	) error

	ListByProductIDs(
		ctx context.Context,
		exec transaction.Executor,
		productIDs []uuid.UUID,
	) (map[uuid.UUID][]productdomain.ProductImage, error)

	ListByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) ([]productdomain.ProductImage, error)

	SoftDeleteByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) error
}

type ProductImageUploadService interface {
	Upload(params UploadProductImagesParams) ([]UploadedProductImage, error)
	Delete(assetKeys []string) error
}

type ProductPerformanceRepository interface {
	UpsertPerformance(
		ctx context.Context,
		exec transaction.Executor,
		perf productdomain.ProductPerformance,
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
	) ([]productdomain.ProductStats, int, error)
}

type ProductStockHistoryRepository interface {
	RecordStockEvent(
		ctx context.Context,
		exec transaction.Executor,
		event productdomain.ProductStockEvent,
	) error
}
