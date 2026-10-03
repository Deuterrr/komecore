package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"

	"github.com/google/uuid"
)

type ProductRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Product, error)

	GetBySlug(
		ctx context.Context,
		exec transaction.Executor,
		string string,
	) (*domain.Product, error)

	FindProducts(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]domain.Product, int, error)

	FindProductsWithInventory(
		ctx context.Context,
		exec transaction.Executor,
		params FindProductParams,
	) ([]domain.ProductWithInventory, int, error)

	FindByIDs(
		ctx context.Context,
		exec transaction.Executor,
		IDs []uuid.UUID,
	) ([]domain.Product, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		product *domain.Product,
	) error

	Delete(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error
}

type ProductImageRepository interface {
	Create(
		ctx context.Context,
		exec transaction.Executor,
		images []domain.ProductImage,
	) error

	ListByProductIDs(
		ctx context.Context,
		exec transaction.Executor,
		productIDs []uuid.UUID,
	) (map[uuid.UUID][]domain.ProductImage, error)

	ListByProductID(
		ctx context.Context,
		exec transaction.Executor,
		productID uuid.UUID,
	) ([]domain.ProductImage, error)

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
		perf domain.ProductPerformance,
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
	) ([]domain.ProductStats, int, error)
}

type ProductStockHistoryRepository interface {
	RecordStockEvent(
		ctx context.Context,
		exec transaction.Executor,
		event domain.ProductStockEvent,
	) error
}
