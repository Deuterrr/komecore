package usecase

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

// ShopProductInfo contains catalog product attributes needed by shop consumers.
type ShopProductInfo struct {
	ID          uuid.UUID
	SKU         string
	Name        string
	Slug        string
	Description *string
	Status      string
	Price       int64
	Weight      *float64
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// ShopProductInventoryInfo contains stock quantities for a shop product.
type ShopProductInventoryInfo struct {
	TotalStock    int
	ReservedStock int
}

func (s ShopProductInventoryInfo) Available() int {
	return s.TotalStock - s.ReservedStock
}

// ShopProductResult pairs a product with its inventory record for a given shop.
type ShopProductResult struct {
	Product   ShopProductInfo
	Inventory ShopProductInventoryInfo
}

// ShopProductProvider decouples the shop module from direct product and inventory repository imports.
type ShopProductProvider interface {
	GetShopProducts(ctx context.Context, exec transaction.Executor, shopID uuid.UUID) ([]ShopProductResult, error)
}

type GetShopProductsUsecase struct {
	provider ShopProductProvider
	executor transaction.Executor
}

func NewGetShopProductsUsecase(
	provider ShopProductProvider,
	executor transaction.Executor,
) *GetShopProductsUsecase {
	return &GetShopProductsUsecase{
		provider: provider,
		executor: executor,
	}
}

func (u *GetShopProductsUsecase) Execute(
	ctx context.Context,
	shopID uuid.UUID,
) ([]ShopProductResult, error) {
	return u.provider.GetShopProducts(ctx, u.executor, shopID)
}
