package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/domain"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"
	shopDomain "komecore/internal/modules/shop/domain"

	"github.com/google/uuid"
)

type CartRepository interface {
	GetWithItemsByCustomerID(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*domain.Cart, error)
	NewCart(ctx context.Context, exec transaction.Executor, customerID uuid.UUID) (*domain.Cart, error)
	Save(ctx context.Context, exec transaction.Executor, cart *domain.Cart) error
}

type ProductCatalogReader interface {
	FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]productDomain.Product, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*productDomain.Product, error)
}

type InventoryStockReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]inventoryDomain.Inventory, error)
	GetByProductIDAndShopID(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID) (*inventoryDomain.Inventory, error)
}

type ProductImageCatalogReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]productDomain.ProductImage, error)
}

type ShopCatalogReader interface {
	FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]shopDomain.Shop, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*shopDomain.Shop, error)
}

const MaxCartItemQuantity = 80

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

type AddItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   domain.ItemOptions
}

type UpdateItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   *domain.ItemOptions
}

type UpdateItemByIDInput struct {
	CustomerID  uuid.UUID
	CartItemID  uuid.UUID
	Quantity    int
	ItemOptions *domain.ItemOptions
}

type RemoveItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	ItemOptions                   *domain.ItemOptions
}

type RemoveItemByIDInput struct {
	CustomerID uuid.UUID
	CartItemID uuid.UUID
}

type CartService struct {
	cartRepo       CartRepository
	inventoryRepo  InventoryStockReader
	productRepo    ProductCatalogReader
	productImgRepo ProductImageCatalogReader
	shopRepo       ShopCatalogReader
	fileStore      storage.Provider
	executor       transaction.Executor
	transactor     transaction.Transactor
}

func NewCartService(
	cartRepo CartRepository,
	inventoryRepo InventoryStockReader,
	productRepo ProductCatalogReader,
	productImgRepo ProductImageCatalogReader,
	shopRepo ShopCatalogReader,
	fileStore storage.Provider,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *CartService {
	return &CartService{
		cartRepo:       cartRepo,
		inventoryRepo:  inventoryRepo,
		productRepo:    productRepo,
		productImgRepo: productImgRepo,
		shopRepo:       shopRepo,
		fileStore:      fileStore,
		executor:       executor,
		transactor:     transactor,
	}
}

// GetCart retrieves the customer cart with hydrated product, inventory, image, and shop details.
func (s *CartService) GetCart(ctx context.Context, customerID uuid.UUID) (*GetCartResult, error) {
	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cart: %w", err)
	}
	if cart == nil {
		cart, err = s.cartRepo.NewCart(ctx, s.executor, customerID)
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
	products, err := s.productRepo.FindByIDs(ctx, s.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart with products: %w", err)
	}

	inventoryMap, err := s.inventoryRepo.ListByProductIDs(ctx, s.executor, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory for cart products: %w", err)
	}

	imagesMap, err := s.productImgRepo.ListByProductIDs(ctx, s.executor, productIDs)
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
		if len(images) > 0 && s.fileStore != nil {
			key := images[0].Variants[productDomain.ResolutionThumbnail].Key
			result.Images.Thumbnail = s.fileStore.PublicURL(key, "public-assets")
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
	if s.shopRepo != nil && len(shopIDs) > 0 {
		shops, err := s.shopRepo.FindByIDs(ctx, s.executor, shopIDs)
		if err == nil {
			for _, sh := range shops {
				shopsMap[sh.ID] = sh
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

// AddItem adds an item to the customer's cart.
func (s *CartService) AddItem(ctx context.Context, input AddItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperrors.NewInvalidInput(domain.ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}
	if input.Quantity >= MaxCartItemQuantity {
		return apperrors.NewBadRequest(fmt.Sprintf("quantity cannot exceed %d", MaxCartItemQuantity))
	}

	shop, err := s.shopRepo.GetByID(ctx, s.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to retrieve shop: %w", err)
	}
	if shop == nil || !shop.IsOperable() {
		return apperrors.NewConflict("shop is currently inactive or not approved for transactions")
	}

	inventory, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inventory == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	product, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to load product with inventory: %w", err)
	}
	if product == nil || product.Status == productDomain.ProductStatusArchived {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}
	if product.Status != productDomain.ProductStatusActive {
		return apperrors.NewConflict(fmt.Sprintf("product %q is currently not available for purchase", product.Name))
	}

	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		cart, err = s.cartRepo.NewCart(ctx, s.executor, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to create cart: %w", err)
		}
	}
	if cart.HasProductInAnotherShop(input.ProductID, input.ShopID) {
		return apperrors.NewConflict(domain.ErrProductAlreadyAssignedToShop.Error())
	}

	targetQuantity := cart.TotalProductQuantity(input.ProductID, input.ShopID) + input.Quantity
	if targetQuantity > inventory.Available() {
		return apperrors.NewConflict(domain.ErrInsufficientStock.Error())
	}

	if err := cart.AddItem(input.ProductID, input.ShopID, input.Quantity, input.ItemOptions); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to add item: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// UpdateItem updates an item quantity in the cart identified by product and shop.
func (s *CartService) UpdateItem(ctx context.Context, input UpdateItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperrors.NewInvalidInput(domain.ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}

	inventory, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inventory == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		return apperrors.NewNotFound(domain.ErrCartNotFound.Error())
	}

	var opts []domain.ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if !cart.HasItem(input.ProductID, input.ShopID, opts...) {
		return apperrors.NewNotFound(domain.ErrCartItemNotFound.Error())
	}

	product, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to retrieve product: %w", err)
	}
	if product == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	totalProductQty := cart.TotalProductQuantity(input.ProductID, input.ShopID)
	if existing := cart.FindItem(input.ProductID, input.ShopID, opts...); existing != nil && existing.DeletedAt == nil {
		totalProductQty = totalProductQty - existing.Quantity + input.Quantity
	} else {
		totalProductQty = totalProductQty + input.Quantity
	}
	if totalProductQty > inventory.Available() {
		return apperrors.NewConflict(domain.ErrInsufficientStock.Error())
	}

	if err := cart.SetItem(input.ProductID, input.ShopID, input.Quantity, opts...); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// UpdateItemByID updates an item in the cart by its CartItemID.
func (s *CartService) UpdateItemByID(ctx context.Context, input UpdateItemByIDInput) error {
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}

	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		return apperrors.NewNotFound(domain.ErrCartNotFound.Error())
	}

	var targetItem *domain.CartItem
	for i := range cart.Items {
		item := &cart.Items[i]
		if item.ID == input.CartItemID && item.DeletedAt == nil {
			targetItem = item
			break
		}
	}
	if targetItem == nil {
		return apperrors.NewNotFound(domain.ErrCartItemNotFound.Error())
	}

	if targetItem.ProductID != uuid.Nil {
		inventory, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, targetItem.ProductID, targetItem.ShopID)
		if err != nil {
			return fmt.Errorf("failed to load inventory by product and shop: %w", err)
		}
		if inventory == nil {
			return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
		}

		product, err := s.productRepo.GetByID(ctx, s.executor, targetItem.ProductID)
		if err != nil {
			return fmt.Errorf("failed to retrieve product: %w", err)
		}
		if product == nil {
			return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
		}

		totalProductQty := cart.TotalProductQuantity(targetItem.ProductID, targetItem.ShopID, targetItem.ID) + input.Quantity
		if totalProductQty > inventory.Available() {
			return apperrors.NewConflict(domain.ErrInsufficientStock.Error())
		}
	}

	var opts []domain.ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if err := cart.UpdateItemByID(input.CartItemID, input.Quantity, opts...); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// RemoveItem removes an item from the cart matching product and shop.
func (s *CartService) RemoveItem(ctx context.Context, input RemoveItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperrors.NewInvalidInput(domain.ErrInvalidShopID.Error())
	}

	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		return apperrors.NewNotFound(domain.ErrCartNotFound.Error())
	}

	var opts []domain.ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if cart.FindItem(input.ProductID, input.ShopID, opts...) == nil {
		return apperrors.NewNotFound(domain.ErrCartItemNotFound.Error())
	}
	cart.RemoveItem(input.ProductID, input.ShopID, opts...)

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to update cart: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// RemoveItemByID removes an item from the cart by its CartItemID.
func (s *CartService) RemoveItemByID(ctx context.Context, input RemoveItemByIDInput) error {
	if input.CartItemID == uuid.Nil {
		return apperrors.NewInvalidInput("invalid cart item id")
	}

	cart, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		return apperrors.NewNotFound(domain.ErrCartNotFound.Error())
	}
	if !cart.RemoveItemByID(input.CartItemID) {
		return apperrors.NewNotFound(domain.ErrCartItemNotFound.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to update cart: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
