package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/repository"

	"github.com/google/uuid"
)

type GetWishlistUsecase struct {
	wishlistRepo   repository.WishlistRepository
	productRepo    productRepo.ProductRepository
	inventoryRepo  inventoryRepo.InventoryRepository
	productImgRepo productRepo.ProductImageRepository
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewGetWishlistUsecase(
	wishlistRepo repository.WishlistRepository,
	productRepo productRepo.ProductRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productImgRepo productRepo.ProductImageRepository,
	fileStore storage.Provider,
	executor transaction.Executor,
) *GetWishlistUsecase {
	return &GetWishlistUsecase{
		wishlistRepo:   wishlistRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		productImgRepo: productImgRepo,
		fileStore:      fileStore,
		executor:       executor,
	}
}

func (u *GetWishlistUsecase) Execute(ctx context.Context, customerID uuid.UUID) ([]domain.WishlistProductView, error) {
	if customerID == uuid.Nil {
		return nil, apperrors.NewBadRequest(domain.ErrInvalidCustomerID.Error())
	}

	items, err := u.wishlistRepo.ListByCustomerID(ctx, u.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wishlist items: %w", err)
	}
	if len(items) == 0 {
		return []domain.WishlistProductView{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := u.productRepo.FindByIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for wishlist: %w", err)
	}
	prodMap := make(map[uuid.UUID]productDomain.Product, len(products))
	for _, p := range products {
		prodMap[p.ID] = p
	}

	var inventoryMap map[uuid.UUID][]inventoryDomain.Inventory
	if u.inventoryRepo != nil {
		invMap, err := u.inventoryRepo.ListByProductIDs(ctx, u.executor, productIDs)
		if err == nil {
			inventoryMap = invMap
		}
	}

	var imagesMap map[uuid.UUID][]productDomain.ProductImage
	if u.productImgRepo != nil {
		imgs, err := u.productImgRepo.ListByProductIDs(ctx, u.executor, productIDs)
		if err == nil {
			imagesMap = imgs
		}
	}

	views := make([]domain.WishlistProductView, 0, len(items))
	for _, item := range items {
		prod, exists := prodMap[item.ProductID]
		if !exists {
			continue
		}

		totalStock := 0
		if inventoryMap != nil {
			if invs, ok := inventoryMap[item.ProductID]; ok {
				for _, inv := range invs {
					totalStock += inv.TotalStock
				}
			}
		}

		var primaryImage *string
		if imagesMap != nil {
			if imgs, ok := imagesMap[item.ProductID]; ok && len(imgs) > 0 {
				if varThumbnail, ok := imgs[0].Variants[productDomain.ResolutionThumbnail]; ok && varThumbnail.Key != "" && u.fileStore != nil {
					url := u.fileStore.PublicURL(varThumbnail.Key, "public-assets")
					primaryImage = &url
				}
			}
		}

		views = append(views, domain.WishlistProductView{
			ProductID:    prod.ID,
			SKU:          prod.SKU,
			Name:         prod.Name,
			Slug:         prod.Slug,
			Price:        prod.Price,
			PrimaryImage: primaryImage,
			InStock:      totalStock > 0 && prod.Status == productDomain.ProductStatusActive,
			TotalStock:   totalStock,
			CreatedAt:    item.CreatedAt,
		})
	}

	return views, nil
}
