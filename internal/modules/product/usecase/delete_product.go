package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type DeleteProductUsecase struct {
	productRepo repository.ProductRepository
	executor    transaction.Executor
	cache       cache.Cache
}

func NewDeleteProductUsecase(
	productRepo repository.ProductRepository,
	executor transaction.Executor,
) *DeleteProductUsecase {
	return &DeleteProductUsecase{
		productRepo: productRepo,
		executor:    executor,
	}
}

func (u *DeleteProductUsecase) WithCache(c cache.Cache) *DeleteProductUsecase {
	u.cache = c
	return u
}

func (u *DeleteProductUsecase) Execute(
	ctx context.Context,
	id uuid.UUID,
) error {
	product, err := u.productRepo.GetByID(ctx, u.executor, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve product: %w", err)
	}
	if product == nil {
		return apperrors.NewNotFound("product not found")
	}

	if err := u.productRepo.Delete(ctx, u.executor, product.ID); err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if u.cache != nil {
		_ = u.cache.DeletePattern(ctx, "cache:product*")
	}

	return nil
}
