package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/domain"
	"komecore/internal/modules/cart/repository"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	productDomain "komecore/internal/modules/product/domain"
	productRepo "komecore/internal/modules/product/repository"
	shopRepo "komecore/internal/modules/shop/repository"

	"github.com/google/uuid"
)

type AddItemUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	cartRepo      repository.CartRepository
	inventoryRepo inventoryRepo.InventoryRepository
	productRepo   productRepo.ProductRepository
	shopRepo      shopRepo.ShopRepository
}

func NewAddItemUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	cartRepo repository.CartRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productRepo productRepo.ProductRepository,
	shopRepo shopRepo.ShopRepository,
) *AddItemUsecase {
	return &AddItemUsecase{
		executor:      executor,
		transactor:    transactor,
		cartRepo:      cartRepo,
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		shopRepo:      shopRepo,
	}
}

type AddItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   domain.ItemOptions
}

const MaxCartItemQuantity = 80

func (u *AddItemUsecase) Execute(ctx context.Context, input AddItemInput) error {
	if input.ShopID == uuid.Nil {
		return apperrors.NewInvalidInput(domain.ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}
	// Added to prevent user from adding too many items at once.
	// Tbh, this is a simple rule after all.
	if input.Quantity >= MaxCartItemQuantity {
		return apperrors.NewBadRequest(fmt.Sprintf("quantity cannot exceed %d", MaxCartItemQuantity))
	}

	shop, err := u.shopRepo.GetByID(ctx, u.executor, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to retrieve shop: %w", err)
	}
	if shop == nil || !shop.IsOperable() {
		return apperrors.NewConflict("shop is currently inactive or not approved for transactions")
	}

	inventory, err := u.inventoryRepo.GetByProductIDAndShopID(ctx, u.executor, input.ProductID, input.ShopID)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inventory == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to load product with inventory: %w", err)
	}
	if product == nil || product.Status == productDomain.ProductStatusArchived {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}
	if product.Status != productDomain.ProductStatusActive {
		return apperrors.NewConflict(fmt.Sprintf("product %q is currently not available for purchase", product.Name))
	}

	cart, err := u.cartRepo.GetWithItemsByCustomerID(ctx, u.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}
	if cart == nil {
		cart, err = u.cartRepo.NewCart(ctx, u.executor, input.CustomerID)
		if err != nil {
			return fmt.Errorf("failed to create cart: %w", err)
		}
	}
	if cart.HasProductInAnotherShop(
		input.ProductID,
		input.ShopID,
	) {
		return apperrors.NewConflict(domain.ErrProductAlreadyAssignedToShop.Error())
	}

	targetQuantity := cart.TotalProductQuantity(input.ProductID, input.ShopID) + input.Quantity
	if targetQuantity > inventory.Available() {
		return apperrors.NewConflict(domain.ErrInsufficientStock.Error())
	}

	if err := cart.AddItem(
		input.ProductID,
		input.ShopID,
		input.Quantity,
		input.ItemOptions,
	); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to add item: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
