package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/common/authctx"
	productRepo "komecore/internal/modules/product/repository"
	"komecore/internal/modules/review/repository"

	"github.com/google/uuid"
)

type DeleteReviewInput struct {
	ReviewID uuid.UUID
	Actor    *authctx.Actor
}

type DeleteReviewUsecase struct {
	reviewRepo  repository.ReviewRepository
	productRepo productRepo.ProductRepository
	cache       cache.Cache
	executor    transaction.Executor
	transactor  transaction.Transactor
}

func NewDeleteReviewUsecase(
	reviewRepo repository.ReviewRepository,
	productRepo productRepo.ProductRepository,
	cache cache.Cache,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *DeleteReviewUsecase {
	return &DeleteReviewUsecase{
		reviewRepo:  reviewRepo,
		productRepo: productRepo,
		cache:       cache,
		executor:    executor,
		transactor:  transactor,
	}
}

func (u *DeleteReviewUsecase) Execute(ctx context.Context, input DeleteReviewInput) error {
	if input.ReviewID == uuid.Nil {
		return apperrors.NewBadRequest("invalid review id")
	}

	review, err := u.reviewRepo.GetByID(ctx, u.executor, input.ReviewID)
	if err != nil {
		return fmt.Errorf("failed to retrieve review: %w", err)
	}
	if review == nil {
		return apperrors.NewNotFound("review not found")
	}

	if input.Actor == nil {
		return apperrors.NewUnauthorized("authentication required")
	}

	switch input.Actor.Type {
	case authctx.AccountTypeCustomer:
		if input.Actor.CustomerID == nil || *input.Actor.CustomerID != review.CustomerID {
			return apperrors.NewForbidden("forbidden: cannot delete review belonging to another customer")
		}
	case authctx.AccountTypeStaff:
		// Staff is authorized to delete/moderate reviews
	default:
		return apperrors.NewForbidden("forbidden: unauthorized to delete reviews")
	}

	err = u.transactor.WithinTransaction(ctx, func(tx transaction.Executor) error {
		if err := u.reviewRepo.Delete(ctx, tx, input.ReviewID); err != nil {
			return err
		}

		summary, err := u.reviewRepo.GetRatingSummary(ctx, tx, review.ProductID)
		if err != nil {
			return err
		}

		if err := u.productRepo.UpdateRating(ctx, tx, review.ProductID, summary.AverageRating, summary.ReviewCount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete review and update rating aggregate: %w", err)
	}

	if u.cache != nil {
		product, _ := u.productRepo.GetByID(ctx, u.executor, review.ProductID)
		if product != nil {
			_ = u.cache.Delete(ctx, fmt.Sprintf("cache:product:slug:%s", product.Slug))
		}
		_ = u.cache.Delete(ctx, "cache:products:list:all")
	}

	return nil
}
