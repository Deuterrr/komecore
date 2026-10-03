package usecase

import (
	"context"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"

	"github.com/google/uuid"
)

type GetShopUsecase struct {
	shopRepo repository.ShopRepository
	executor transaction.Executor
}

func NewGetShopUsecase(
	shopRepo repository.ShopRepository,
	executor transaction.Executor,
) *GetShopUsecase {
	return &GetShopUsecase{
		shopRepo: shopRepo,
		executor: executor,
	}
}

func (u *GetShopUsecase) GetByID(
	ctx context.Context,
	shopID uuid.UUID,
) (*domain.Shop, error) {
	shop, err := u.shopRepo.GetByID(ctx, u.executor, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop: %w", err)
	}

	return shop, nil
}

func (u *GetShopUsecase) GetBySlug(
	ctx context.Context,
	slug string,
) (*domain.Shop, error) {
	shop, err := u.shopRepo.GetBySlug(ctx, u.executor, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shop by slug: %w", err)
	}

	return shop, nil
}
