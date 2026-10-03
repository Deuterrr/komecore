package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/repository"

	"github.com/google/uuid"
)

type DeleteProductUsecase struct {
	productRepo repository.ProductRepository
	executor    transaction.Executor
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

	return nil
}
