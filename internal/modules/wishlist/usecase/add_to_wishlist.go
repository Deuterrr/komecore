package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	productRepo "komecore/internal/modules/product/repository"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/repository"

	"github.com/google/uuid"
)

type AddToWishlistUsecase struct {
	wishlistRepo repository.WishlistRepository
	productRepo  productRepo.ProductRepository
	executor     transaction.Executor
}

func NewAddToWishlistUsecase(
	wishlistRepo repository.WishlistRepository,
	productRepo productRepo.ProductRepository,
	executor transaction.Executor,
) *AddToWishlistUsecase {
	return &AddToWishlistUsecase{
		wishlistRepo: wishlistRepo,
		productRepo:  productRepo,
		executor:     executor,
	}
}

type AddToWishlistInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
}

func (u *AddToWishlistUsecase) Execute(ctx context.Context, input AddToWishlistInput) error {
	if input.CustomerID == uuid.Nil {
		return apperrors.NewBadRequest(domain.ErrInvalidCustomerID.Error())
	}
	if input.ProductID == uuid.Nil {
		return apperrors.NewBadRequest(domain.ErrInvalidProductID.Error())
	}

	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check product existence: %w", err)
	}
	if product == nil {
		return apperrors.NewNotFound(domain.ErrProductNotFound.Error())
	}

	exists, err := u.wishlistRepo.Exists(ctx, u.executor, input.CustomerID, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check existing wishlist item: %w", err)
	}
	if exists {
		return apperrors.NewConflict(domain.ErrWishlistItemAlreadyExists.Error())
	}

	item := domain.WishlistItem{
		CustomerID: input.CustomerID,
		ProductID:  input.ProductID,
		CreatedAt:  time.Now(),
	}

	if err := u.wishlistRepo.Add(ctx, u.executor, item); err != nil {
		return fmt.Errorf("failed to add item to wishlist: %w", err)
	}

	return nil
}
