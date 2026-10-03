package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
)

type ListShopAddressesUsecase struct {
	shopAddressRepo repository.ShopAddressRepository
	executor        transaction.Executor
}

func NewListShopAddressesUsecase(
	shopAddressRepo repository.ShopAddressRepository,
	executor transaction.Executor,
) *ListShopAddressesUsecase {
	return &ListShopAddressesUsecase{
		shopAddressRepo: shopAddressRepo,
		executor:        executor,
	}
}

func (u *ListShopAddressesUsecase) FindByShopID(ctx context.Context, shopID uuid.UUID) ([]domain.ShopAddress, error) {
	res, err := u.shopAddressRepo.FindByShopID(ctx, u.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve address: %w", err)
	}

	return res, nil
}
