package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	addressDomain "komecore/internal/modules/address/domain"
	addressRepo "komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type GetShopAddressesUsecase struct {
	addressRepo addressRepo.ShopAddressRepository
	executor    transaction.Executor
}

func NewGetShopAddressesUsecase(
	addressRepo addressRepo.ShopAddressRepository,
	executor transaction.Executor,
) *GetShopAddressesUsecase {
	return &GetShopAddressesUsecase{
		addressRepo: addressRepo,
		executor:    executor,
	}
}

func (u *GetShopAddressesUsecase) Execute(
	ctx context.Context,
	shopID uuid.UUID,
) ([]addressDomain.ShopAddress, error) {
	addresses, err := u.addressRepo.FindByShopID(ctx, u.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop addresses: %w", err)
	}

	return addresses, nil
}
