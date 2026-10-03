package usecase

import (
	"context"
	"fmt"

	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/domain"
	"komecore/internal/modules/cart/repository"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"
	shopDomain "komecore/internal/modules/shop/domain"
	shopRepo "komecore/internal/modules/shop/repository"

	"github.com/google/uuid"
)

type GetCartUsecase struct {
	cartRepo       repository.CartRepository
	inventoryRepo  inventoryRepo.InventoryRepository
	productRepo    productRepo.ProductRepository
	productImgRepo productRepo.ProductImageRepository
	shopRepo       shopRepo.ShopRepository
	fileStore      storage.Provider
	executor       transaction.Executor
}

func NewGetCartUsecase(
	cartRepo repository.CartRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productRepo productRepo.ProductRepository,
	productImgRepo productRepo.ProductImageRepository,
	shopRepo shopRepo.ShopRepository,
	fileStore storage.Provider,
	executor transaction.Executor,
) *GetCartUsecase {
	return &GetCartUsecase{
		cartRepo:       cartRepo,
		inventoryRepo:  inventoryRepo,
		productRepo:    productRepo,
		productImgRepo: productImgRepo,
		shopRepo:       shopRepo,
		fileStore:      fileStore,
		executor:       executor,
	}
}

type ProductCartResponse struct {
	Product   productDomain.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventoryDomain.Inventory
	Images          struct {
		Thumbnail string
	}
}

type GetCartResult struct {
	Cart     *domain.Cart
	Products map[uuid.UUID]ProductCartResponse
	Shops    map[uuid.UUID]shopDomain.Shop
}

func (u *GetCartUsecase) Execute(ctx context.Context, customerID uuid.UUID) (*GetCartResult, error) {
	cart, err := u.cartRepo.GetWithItemsByCustomerID(ctx, u.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cart: %w", err)
	}
	if cart == nil {
		cart, err = u.cartRepo.NewCart(ctx, u.executor, customerID)
		if err != nil {
			return nil, fmt.Errorf("failed to create cart: %w", err)
		}
	}
	if len(cart.Items) == 0 {
		return &GetCartResult{
			Cart:     cart,
			Products: map[uuid.UUID]ProductCartResponse{},
		}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(cart.Items))
	for _, item := range cart.Items {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := u.productRepo.FindByIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart with products: %w", err)
	}

	inventoryMap, err := u.inventoryRepo.ListByProductIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory for cart products: %w", err)
	}

	imagesMap, err := u.productImgRepo.ListByProductIDs(ctx, u.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load images for products: %w", err)
	}

	productMap := make(map[uuid.UUID]ProductCartResponse)
	for _, p := range products {
		inventories := inventoryMap[p.ID]
		result := ProductCartResponse{
			Product:         p,
			ShopInventories: inventories,
		}

		images := imagesMap[p.ID]
		if len(images) > 0 {
			key := images[0].Variants[productDomain.ResolutionThumbnail].Key
			result.Images.Thumbnail = u.fileStore.PublicURL(key, "public-assets")
		}

		for _, inventory := range inventories {
			result.Inventory.TotalStock += inventory.TotalStock
			result.Inventory.ReservedStock += inventory.ReservedStock
		}

		productMap[p.ID] = result
	}

	activeItems := make([]domain.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if _, ok := productMap[item.ProductID]; ok {
			activeItems = append(activeItems, item)
		}
	}
	cart.Items = activeItems

	shopIDsMap := make(map[uuid.UUID]bool)
	for _, item := range cart.Items {
		if item.ShopID != uuid.Nil {
			shopIDsMap[item.ShopID] = true
		}
	}
	shopIDs := make([]uuid.UUID, 0, len(shopIDsMap))
	for id := range shopIDsMap {
		shopIDs = append(shopIDs, id)
	}
	shopsMap := make(map[uuid.UUID]shopDomain.Shop)
	if u.shopRepo != nil && len(shopIDs) > 0 {
		shops, err := u.shopRepo.FindByIDs(ctx, u.executor, shopIDs)
		if err == nil {
			for _, s := range shops {
				shopsMap[s.ID] = s
			}
		}
	}

	result := GetCartResult{
		Cart:     cart,
		Products: productMap,
		Shops:    shopsMap,
	}

	return &result, nil
}
