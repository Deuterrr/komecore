package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type DeleteShopAddressUsecase struct {
	shopAddressRepo repository.ShopAddressRepository
	executor        transaction.Executor
}

func NewDeleteShopAddressUsecase(
	shopAddressRepo repository.ShopAddressRepository,
	executor transaction.Executor,
) *DeleteShopAddressUsecase {
	return &DeleteShopAddressUsecase{
		shopAddressRepo: shopAddressRepo,
		executor:        executor,
	}
}

func (u *DeleteShopAddressUsecase) Execute(ctx context.Context, shopID uuid.UUID, addressID uuid.UUID) error {
	address, err := u.shopAddressRepo.GetByID(ctx, u.executor, addressID)
	if err != nil {
		return fmt.Errorf("failed to retrieve address: %w", err)
	}
	if address == nil || address.ShopID != shopID {
		return apperrors.NewNotFound(domain.ErrAddressNotFound.Error())
	}
	if address.IsActive {
		return apperrors.NewConflict(domain.ErrCannotDeleteDefaultAddress.Error())
	}

	if err := u.shopAddressRepo.Delete(ctx, u.executor, addressID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}
