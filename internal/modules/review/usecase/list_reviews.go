package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	productRepo "komecore/internal/modules/product/repository"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/repository"

	"github.com/google/uuid"
)

type ListReviewsInput struct {
	ProductID uuid.UUID
	Page      int
	Limit     int
}

type ListReviewsResult struct {
	Reviews       []domain.ReviewWithCustomer
	Total         int
	Page          int
	Limit         int
	AverageRating float64
	ReviewCount   int
}

type ListReviewsUsecase struct {
	reviewRepo  repository.ReviewRepository
	productRepo productRepo.ProductRepository
	executor    transaction.Executor
}

func NewListReviewsUsecase(
	reviewRepo repository.ReviewRepository,
	productRepo productRepo.ProductRepository,
	executor transaction.Executor,
) *ListReviewsUsecase {
	return &ListReviewsUsecase{
		reviewRepo:  reviewRepo,
		productRepo: productRepo,
		executor:    executor,
	}
}

func (u *ListReviewsUsecase) Execute(ctx context.Context, input ListReviewsInput) (*ListReviewsResult, error) {
	if input.ProductID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invalid product id")
	}

	product, err := u.productRepo.GetByID(ctx, u.executor, input.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve product: %w", err)
	}
	if product == nil {
		return nil, apperrors.NewNotFound("product not found")
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	reviews, total, err := u.reviewRepo.ListByProductID(ctx, u.executor, repository.ListReviewsParams{
		ProductID: input.ProductID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list product reviews: %w", err)
	}

	summary, err := u.reviewRepo.GetRatingSummary(ctx, u.executor, input.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating summary: %w", err)
	}

	return &ListReviewsResult{
		Reviews:       reviews,
		Total:         total,
		Page:          page,
		Limit:         limit,
		AverageRating: summary.AverageRating,
		ReviewCount:   summary.ReviewCount,
	}, nil
}
