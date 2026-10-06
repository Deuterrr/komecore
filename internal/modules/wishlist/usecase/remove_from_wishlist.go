package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/repository"

	"github.com/google/uuid"
)

type RemoveFromWishlistUsecase struct {
	wishlistRepo repository.WishlistRepository
	executor     transaction.Executor
}

func NewRemoveFromWishlistUsecase(
	wishlistRepo repository.WishlistRepository,
	executor transaction.Executor,
) *RemoveFromWishlistUsecase {
	return &RemoveFromWishlistUsecase{
		wishlistRepo: wishlistRepo,
		executor:     executor,
	}
}

type RemoveFromWishlistInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
}

func (u *RemoveFromWishlistUsecase) Execute(ctx context.Context, input RemoveFromWishlistInput) error {
	if input.CustomerID == uuid.Nil {
		return apperrors.NewBadRequest(domain.ErrInvalidCustomerID.Error())
	}
	if input.ProductID == uuid.Nil {
		return apperrors.NewBadRequest(domain.ErrInvalidProductID.Error())
	}

	exists, err := u.wishlistRepo.Exists(ctx, u.executor, input.CustomerID, input.ProductID)
	if err != nil {
		return fmt.Errorf("failed to check existing wishlist item: %w", err)
	}
	if !exists {
		return apperrors.NewNotFound(domain.ErrWishlistItemNotFound.Error())
	}

	if err := u.wishlistRepo.Remove(ctx, u.executor, input.CustomerID, input.ProductID); err != nil {
		return fmt.Errorf("failed to remove item from wishlist: %w", err)
	}

	return nil
}
