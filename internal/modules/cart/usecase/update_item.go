package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/cart/domain"
	"komecore/internal/modules/cart/repository"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	productRepo "komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type UpdateItemUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	cartRepo      repository.CartRepository
	inventoryRepo inventoryRepo.InventoryRepository
	productRepo   productRepo.ProductRepository
}

func NewUpdateItemUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	cartRepo repository.CartRepository,
	inventoryRepo inventoryRepo.InventoryRepository,
	productRepo productRepo.ProductRepository,
) *UpdateItemUsecase {
	return &UpdateItemUsecase{
		executor:      executor,
		transactor:    transactor,
		cartRepo:      cartRepo,
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
	}
}

type UpdateItemInput struct {
	CustomerID, ProductID, ShopID uuid.UUID
	Quantity                      int
	ItemOptions                   *domain.ItemOptions
}

func (u *UpdateItemUsecase) Execute(
	ctx context.Context,
	input UpdateItemInput,
) error {
	if input.ShopID == uuid.Nil {
		return apperrors.NewInvalidInput(domain.ErrInvalidShopID.Error())
	}
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}

	inventory, err := u.inventoryRepo.GetByProductIDAndShopID(ctx, u.executor,
		input.ProductID,
		input.ShopID,
	)
	if err != nil {
		return fmt.Errorf("failed to load inventory by product and shop: %w", err)
	}
	if inventory == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	cart, err := u.cartRepo.GetWithItemsByCustomerID(ctx, u.executor, input.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to load cart with items: %w", err)
	}

	// The updated implementation removes creates a cart when none exists
	// instead returns ErrCartNotFound directly.
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

	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
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

	if err := cart.SetItem(
		input.ProductID,
		input.ShopID,
		input.Quantity,
		opts...,
	); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	if err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.cartRepo.Save(ctx, exec, cart); err != nil {
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
	ItemOptions *domain.ItemOptions
}

func (u *UpdateItemUsecase) ExecuteByID(ctx context.Context, input UpdateItemByIDInput) error {
	if input.Quantity <= 0 {
		return apperrors.NewInvalidInput(domain.ErrInvalidQuantity.Error())
	}

	cart, err := u.cartRepo.GetWithItemsByCustomerID(ctx, u.executor, input.CustomerID)
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
		inventory, err := u.inventoryRepo.GetByProductIDAndShopID(ctx, u.executor,
			targetItem.ProductID,
			targetItem.ShopID,
		)
		if err != nil {
			return fmt.Errorf("failed to load inventory by product and shop: %w", err)
		}
		if inventory == nil {
			return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
		}

		product, err := u.productRepo.GetByID(ctx, u.executor, targetItem.ProductID)
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

	if err := cart.UpdateItemByID(
		input.CartItemID,
		input.Quantity,
		opts...,
	); err != nil {
		return apperrors.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.cartRepo.Save(ctx, exec, cart); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
