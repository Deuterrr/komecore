package cart

import (
	"context"
	"errors"
	"fmt"

	"komecore/internal/apperror"
	"komecore/internal/infra/storage"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/shop"

	"github.com/google/uuid"
)

type ProductCatalogReader interface {
	FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]product.Product, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*product.Product, error)
}

type InventoryStockReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error)
	GetByProductIDAndShopID(ctx context.Context, exec transaction.Executor, productID, shopID uuid.UUID) (*inventory.Inventory, error)
}

type ProductImageCatalogReader interface {
	ListByProductIDs(ctx context.Context, exec transaction.Executor, productIDs []uuid.UUID) (map[uuid.UUID][]product.ProductImage, error)
}

type ShopCatalogReader interface {
	FindByIDs(ctx context.Context, exec transaction.Executor, ids []uuid.UUID) ([]shop.Shop, error)
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*shop.Shop, error)
}

const MaxCartItemQuantity = 80

type Service struct {
	cartRepo       Repository
	inventoryRepo  InventoryStockReader
	productRepo    ProductCatalogReader
	productImgRepo ProductImageCatalogReader
	shopRepo       ShopCatalogReader
	fileStore      storage.Provider
	executor       transaction.Executor
	transactor     transaction.Transactor
}

func NewService(
	cartRepo Repository,
	inventoryRepo InventoryStockReader,
	productRepo ProductCatalogReader,
	productImgRepo ProductImageCatalogReader,
	shopRepo ShopCatalogReader,
	fileStore storage.Provider,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *Service {
	return &Service{
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

type ProductCartResponse struct {
	Product   product.Product
	Inventory struct {
		TotalStock    int
		ReservedStock int
	}
	ShopInventories []inventory.Inventory
	Images          struct {
		Thumbnail string
	}
}

type GetCartResult struct {
	Cart     *Cart
	Products map[uuid.UUID]ProductCartResponse
	Shops    map[uuid.UUID]shop.Shop
}

// GetCart retrieves the customer cart with hydrated product, inventory, image, and shop details.
func (s *Service) GetCart(ctx context.Context, customerID uuid.UUID) (*GetCartResult, error) {
	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cart: %w", err)
	}
	if c == nil {
		c, err = s.cartRepo.NewCart(ctx, s.executor, customerID)
		if err != nil {
			return nil, fmt.Errorf("failed to create cart: %w", err)
		}
	}
	if len(c.Items) == 0 {
		return &GetCartResult{
			Cart:     c,
			Products: map[uuid.UUID]ProductCartResponse{},
		}, nil
	}

	productIDs := make([]uuid.UUID, 0, len(c.Items))
	for _, item := range c.Items {
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
			key := images[0].Variants[product.ResolutionThumbnail].Key
			result.Images.Thumbnail = s.fileStore.PublicURL(key, "public-assets")
		}

		for _, invItem := range inventories {
			result.Inventory.TotalStock += invItem.TotalStock
			result.Inventory.ReservedStock += invItem.ReservedStock
		}

		productMap[p.ID] = result
	}

	activeItems := make([]CartItem, 0, len(c.Items))
	for _, item := range c.Items {
		if _, ok := productMap[item.ProductID]; ok {
			activeItems = append(activeItems, item)
		}
	}
	c.Items = activeItems

	shopIDsMap := make(map[uuid.UUID]bool)
	for _, item := range c.Items {
		if item.ShopID != uuid.Nil {
			shopIDsMap[item.ShopID] = true
		}
	}
	shopIDs := make([]uuid.UUID, 0, len(shopIDsMap))
	for id := range shopIDsMap {
		shopIDs = append(shopIDs, id)
	}
	shopsMap := make(map[uuid.UUID]shop.Shop)
	if s.shopRepo != nil && len(shopIDs) > 0 {
		shops, err := s.shopRepo.FindByIDs(ctx, s.executor, shopIDs)
		if err == nil {
			for _, sh := range shops {
				shopsMap[sh.ID] = sh
			}
		}
	}

	result := GetCartResult{
		Cart:     c,
		Products: productMap,
		Shops:    shopsMap,
	}
	return &result, nil
}

type AddItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   ItemOptions
}

// AddItem adds an item to the customer's cart.
func (s *Service) AddItem(ctx context.Context, input AddItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperror.NewInvalidInput(ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperror.NewInvalidInput(ErrInvalidQuantity.Error())
	}
	if input.Quantity >= MaxCartItemQuantity {
		return apperror.NewBadRequest(fmt.Sprintf("quantity cannot exceed %d", MaxCartItemQuantity))
	}

	shp, err := s.shopRepo.GetByID(ctx, s.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to retrieve shop: %w", err)
	}
	if shp == nil || !shp.IsOperable() {
		return apperror.NewConflict("shop is currently inactive or not approved for transactions")
	}

	inv, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inv == nil {
		return apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, ErrProductNotFound.Error())
	}

	prod, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to load product with inventory: %w", err)
	}
	if prod == nil || prod.Status == product.ProductStatusArchived {
		return apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, ErrProductNotFound.Error())
	}
	if prod.Status != product.ProductStatusActive {
		return apperror.NewConflict(fmt.Sprintf("product %q is currently not available for purchase", prod.Name))
	}

	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if c == nil {
		c, err = s.cartRepo.NewCart(ctx, s.executor, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to create cart: %w", err)
		}
	}
	if c.HasProductInAnotherShop(input.ProductID, input.ShopID) {
		return apperror.NewConflict(ErrProductAlreadyAssignedToShop.Error())
	}

	targetQuantity := c.TotalProductQuantity(input.ProductID, input.ShopID) + input.Quantity
	if targetQuantity > inv.Available() {
		return apperror.NewConflictWithCode(apperror.CodeInsufficientStock, ErrInsufficientStock.Error())
	}

	if err := c.AddItem(input.ProductID, input.ShopID, input.Quantity, input.ItemOptions); err != nil {
		return apperror.NewInvalidInput(err.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, c); err != nil {
			return fmt.Errorf("failed to add item: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(transaction.MapDBError(err), transaction.ErrUniqueViolation) {
			return apperror.NewConflict("item already exists in cart or concurrently modified")
		}
		return err
	}

	return nil
}

type UpdateItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   *ItemOptions
}

// UpdateItem updates an item quantity in the cart identified by product and shop.
func (s *Service) UpdateItem(ctx context.Context, input UpdateItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperror.NewInvalidInput(ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperror.NewInvalidInput(ErrInvalidQuantity.Error())
	}

	inv, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inv == nil {
		return apperror.NewNotFound(ErrProductNotFound.Error())
	}

	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if c == nil {
		return apperror.NewNotFound(ErrCartNotFound.Error())
	}

	var opts []ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if !c.HasItem(input.ProductID, input.ShopID, opts...) {
		return apperror.NewNotFoundWithCode(apperror.CodeCartItemNotFound, ErrCartItemNotFound.Error())
	}

	prod, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to retrieve product: %w", err)
	}
	if prod == nil {
		return apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, ErrProductNotFound.Error())
	}

	totalProductQty := c.TotalProductQuantity(input.ProductID, input.ShopID)
	if existing := c.FindItem(input.ProductID, input.ShopID, opts...); existing != nil && existing.DeletedAt == nil {
		totalProductQty = totalProductQty - existing.Quantity + input.Quantity
	} else {
		totalProductQty = totalProductQty + input.Quantity
	}
	if totalProductQty > inv.Available() {
		return apperror.NewConflictWithCode(apperror.CodeInsufficientStock, ErrInsufficientStock.Error())
	}

	if err := c.SetItem(input.ProductID, input.ShopID, input.Quantity, opts...); err != nil {
		return apperror.NewInvalidInput(err.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, c); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

type UpdateItemByIDInput struct {
	CustomerID  uuid.UUID
	CartItemID  uuid.UUID
	Quantity    int
	ItemOptions *ItemOptions
}

// UpdateItemByID updates an item in the cart by its CartItemID.
func (s *Service) UpdateItemByID(ctx context.Context, input UpdateItemByIDInput) error {
	if input.Quantity <= 0 {
		return apperror.NewInvalidInput(ErrInvalidQuantity.Error())
	}

	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if c == nil {
		return apperror.NewNotFound(ErrCartNotFound.Error())
	}

	var targetItem *CartItem
	for i := range c.Items {
		item := &c.Items[i]
		if item.ID == input.CartItemID && item.DeletedAt == nil {
			targetItem = item
			break
		}
	}
	if targetItem == nil {
		return apperror.NewNotFoundWithCode(apperror.CodeCartItemNotFound, ErrCartItemNotFound.Error())
	}

	if targetItem.ProductID != uuid.Nil {
		inv, err := s.inventoryRepo.GetByProductIDAndShopID(ctx, s.executor, targetItem.ProductID, targetItem.ShopID)
		if err != nil {
			return fmt.Errorf("failed to load inventory by product and shop: %w", err)
		}
		if inv == nil {
			return apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, ErrProductNotFound.Error())
		}

		prod, err := s.productRepo.GetByID(ctx, s.executor, targetItem.ProductID)
		if err != nil {
			return fmt.Errorf("failed to retrieve product: %w", err)
		}
		if prod == nil {
			return apperror.NewNotFoundWithCode(apperror.CodeProductNotFound, ErrProductNotFound.Error())
		}

		totalProductQty := c.TotalProductQuantity(targetItem.ProductID, targetItem.ShopID, targetItem.ID) + input.Quantity
		if totalProductQty > inv.Available() {
			return apperror.NewConflictWithCode(apperror.CodeInsufficientStock, ErrInsufficientStock.Error())
		}
	}

	var opts []ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if err := c.UpdateItemByID(input.CartItemID, input.Quantity, opts...); err != nil {
		return apperror.NewInvalidInput(err.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, c); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

type RemoveItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	ItemOptions                   *ItemOptions
}

// RemoveItem removes an item from the cart matching product and shop.
func (s *Service) RemoveItem(ctx context.Context, input RemoveItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperror.NewInvalidInput(ErrInvalidShopID.Error())
	}

	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if c == nil {
		return apperror.NewNotFound(ErrCartNotFound.Error())
	}

	var opts []ItemOptions
	if input.ItemOptions != nil {
		opts = append(opts, *input.ItemOptions)
	}

	if c.FindItem(input.ProductID, input.ShopID, opts...) == nil {
		return apperror.NewNotFoundWithCode(apperror.CodeCartItemNotFound, ErrCartItemNotFound.Error())
	}
	c.RemoveItem(input.ProductID, input.ShopID, opts...)

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, c); err != nil {
			return fmt.Errorf("failed to update cart: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

type RemoveItemByIDInput struct {
	CustomerID uuid.UUID
	CartItemID uuid.UUID
}

// RemoveItemByID removes an item from the cart by its CartItemID.
func (s *Service) RemoveItemByID(ctx context.Context, input RemoveItemByIDInput) error {
	if input.CartItemID == uuid.Nil {
		return apperror.NewInvalidInput("invalid cart item id")
	}

	c, err := s.cartRepo.GetWithItemsByCustomerID(ctx, s.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if c == nil {
		return apperror.NewNotFound(ErrCartNotFound.Error())
	}
	if !c.RemoveItemByID(input.CartItemID) {
		return apperror.NewNotFoundWithCode(apperror.CodeCartItemNotFound, ErrCartItemNotFound.Error())
	}

	if err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.cartRepo.Save(ctx, exec, c); err != nil {
			return fmt.Errorf("failed to update cart: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
