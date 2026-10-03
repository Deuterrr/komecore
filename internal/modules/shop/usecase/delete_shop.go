package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	authorDomain "komecore/internal/modules/auth/domain"
	"komecore/internal/modules/shop/repository"

	"github.com/google/uuid"
)

type DeleteShopUsecase struct {
	shopRepo repository.ShopRepository
	executor transaction.Executor
}

func NewDeleteShopUsecase(
	shopRepo repository.ShopRepository,
	executor transaction.Executor,
) *DeleteShopUsecase {
	return &DeleteShopUsecase{
		shopRepo: shopRepo,
		executor: executor,
	}
}

func (u *DeleteShopUsecase) Execute(
	ctx context.Context,
	actor authorDomain.Actor,
	shopID uuid.UUID,
) error {
	isAdmin := false
	for _, role := range actor.Roles {
		if role.Code == authorDomain.RoleStaffAdmin {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return apperrors.NewForbidden("insufficient permissions to delete shop")
	}

	shop, err := u.shopRepo.GetByID(ctx, u.executor, shopID)
	if err != nil {
		return fmt.Errorf("failed to retrieve shop: %w", err)
	}
	if shop == nil {
		return apperrors.NewNotFound("shop not found")
	}

	if err := u.shopRepo.Delete(ctx, u.executor, shop.ID); err != nil {
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	return nil
}
