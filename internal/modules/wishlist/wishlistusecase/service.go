package wishlistusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/wishlist"

	"github.com/google/uuid"
)

type WishlistRepository interface {
	Add(ctx context.Context, exec transaction.Executor, item wishlist.WishlistItem) error
	Remove(ctx context.Context, exec transaction.Executor, customerID, productID uuid.UUID) error
	ListByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) ([]wishlist.WishlistItem, error)
	Exists(ctx context.Context, exec transaction.Executor, customerID, productID uuid.UUID) (bool, error)
}

type ProductReader interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*product.Product, error)
	FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]product.Product, error)
}

type InventoryReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error)
}

type ProductImageReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]product.ProductImage, error)
}

type AddToWishlistInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
}

type RemoveFromWishlistInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
}

type WishlistService struct {
	wishlistRepo   WishlistRepository
	productRepo    ProductReader
	inventoryRepo  InventoryReader
	productImgRepo ProductImageReader
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewWishlistService(
	wishlistRepo WishlistRepository,
	productRepo ProductReader,
	inventoryRepo InventoryReader,
	productImgRepo ProductImageReader,
	fileStore storage.Provider,
	executor transaction.Executor,
) *WishlistService {
	return &WishlistService{
		wishlistRepo:   wishlistRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		productImgRepo: productImgRepo,
		fileStore:      fileStore,
		executor:       executor,
	}
}

func (s *WishlistService) GetWishlist(ctx context.Context, customerID uuid.UUID) ([]wishlist.WishlistProductView, error) {
	if customerID == uuid.Nil {
		return nil, apperror.NewBadRequest(wishlist.ErrInvalidCustomerID.Error())
	}

	items, err := s.wishlistRepo.ListByCustomerID(ctx, s.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list wishlist items: %w", err)
	}
	if len(items) == 0 {
		return []wishlist.WishlistProductView{}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := s.productRepo.FindByIDs(ctx, s.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products for wishlist: %w", err)
	}
	prodMap := make(map[uuid.UUID]product.Product, len(products))
	for _, p := range products {
		prodMap[p.ID] = p
	}

	var inventoryMap map[uuid.UUID][]inventory.Inventory
	if s.inventoryRepo != nil {
		invMap, err := s.inventoryRepo.ListByProductIDs(ctx, s.executor, productIDs)
		if err == nil {
			inventoryMap = invMap
		}
	}

	var imagesMap map[uuid.UUID][]product.ProductImage
	if s.productImgRepo != nil {
		imgs, err := s.productImgRepo.ListByProductIDs(ctx, s.executor, productIDs)
		if err == nil {
			imagesMap = imgs
		}
	}

	views := make([]wishlist.WishlistProductView, 0, len(items))
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
				if varThumbnail, ok := imgs[0].Variants[product.ResolutionThumbnail]; ok && varThumbnail.Key != "" && s.fileStore != nil {
					url := s.fileStore.PublicURL(varThumbnail.Key, "public-assets")
					primaryImage = &url
				}
			}
		}

		views = append(views, wishlist.WishlistProductView{
			ProductID:    prod.ID,
			SKU:          prod.SKU,
			Name:         prod.Name,
			Slug:         prod.Slug,
			Price:        prod.Price,
			PrimaryImage: primaryImage,
			InStock:      totalStock > 0 && prod.Status == product.ProductStatusActive,
			TotalStock:   totalStock,
			CreatedAt:    item.CreatedAt,
		})
	}

	return views, nil
}

func (s *WishlistService) AddToWishlist(ctx context.Context, input AddToWishlistInput) error {
	if input.CustomerID == uuid.Nil {
		return apperror.NewBadRequest(wishlist.ErrInvalidCustomerID.Error())
	}
	if input.ProductID == uuid.Nil {
		return apperror.NewBadRequest(wishlist.ErrInvalidProductID.Error())
	}

	product, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check product existence: %w", err)
	}
	if product == nil {
		return apperror.NewNotFound(wishlist.ErrProductNotFound.Error())
	}

	exists, err := s.wishlistRepo.Exists(ctx, s.executor, input.CustomerID, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check existing wishlist item: %w", err)
	}
	if exists {
		return apperror.NewConflict(wishlist.ErrWishlistItemAlreadyExists.Error())
	}

	item := wishlist.WishlistItem{
		CustomerID: input.CustomerID,
		ProductID:  input.ProductID,
		CreatedAt:  time.Now(),
	}

	if err := s.wishlistRepo.Add(ctx, s.executor, item); err != nil {
		return fmt.Errorf("failed to add item to wishlist: %w", err)
	}

	return nil
}

func (s *WishlistService) RemoveFromWishlist(ctx context.Context, input RemoveFromWishlistInput) error {
	if input.CustomerID == uuid.Nil {
		return apperror.NewBadRequest(wishlist.ErrInvalidCustomerID.Error())
	}
	if input.ProductID == uuid.Nil {
		return apperror.NewBadRequest(wishlist.ErrInvalidProductID.Error())
	}

	exists, err := s.wishlistRepo.Exists(ctx, s.executor, input.CustomerID, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check existing wishlist item: %w", err)
	}
	if !exists {
		return apperror.NewNotFound(wishlist.ErrWishlistItemNotFound.Error())
	}

	if err := s.wishlistRepo.Remove(ctx, s.executor, input.CustomerID, input.ProductID); err != nil {
		return fmt.Errorf("failed to remove item from wishlist: %w", err)
	}

	return nil
}
