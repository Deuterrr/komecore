package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	productRepo "komecore/internal/modules/product/repository"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type CreateReviewInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
	OrderID    *uuid.UUID
	Rating     int
	Title      *string
	Comment    *string
}

type CreateReviewUsecase struct {
	reviewRepo    repository.ReviewRepository
	productRepo   productRepo.ProductRepository
	orderRepo     orderRepo.OrderRepository
	orderItemRepo orderRepo.OrderItemRepository
	cache         cache.Cache
	executor      transaction.Executor
	transactor    transaction.Transactor
}

func NewCreateReviewUsecase(
	reviewRepo repository.ReviewRepository,
	productRepo productRepo.ProductRepository,
	orderRepo orderRepo.OrderRepository,
	orderItemRepo orderRepo.OrderItemRepository,
	cache cache.Cache,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *CreateReviewUsecase {
	return &CreateReviewUsecase{
		reviewRepo:    reviewRepo,
		productRepo:   productRepo,
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		cache:         cache,
		executor:      executor,
		transactor:    transactor,
	}
}

func (u *CreateReviewUsecase) Execute(ctx context.Context, input CreateReviewInput) (*domain.Review, error) {
	if input.Rating < 1 || input.Rating > 5 {
		return nil, apperrors.NewBadRequest(domain.ErrInvalidRating.Error())
	}
	if input.CustomerID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invalid customer id")
	}
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

	var targetOrderID uuid.UUID

	if input.OrderID != nil {
		order, err := u.orderRepo.GetByID(ctx, u.executor, *input.OrderID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve order: %w", err)
		}
		if order == nil || order.CustomerID != input.CustomerID {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}
		if order.Status != orderDomain.OrderStatusDelivered && string(order.Status) != "completed" {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}

		items, err := u.orderItemRepo.ListByOrderID(ctx, u.executor, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}
		hasProduct := false
		for _, it := range items {
			if it.ProductID == input.ProductID {
				hasProduct = true
				break
			}
		}
		if !hasProduct {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}

		hasReviewed, err := u.reviewRepo.HasReviewedOrder(ctx, u.executor, input.CustomerID, input.ProductID, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check order review status: %w", err)
		}
		if hasReviewed {
			return nil, apperrors.NewConflict(domain.ErrDuplicateReview.Error())
		}

		targetOrderID = order.ID
	} else {
		orders, _, err := u.orderRepo.FindOrders(ctx, u.executor, orderRepo.FindOrderParams{
			CustomerID: &input.CustomerID,
			Statuses:   []string{"delivered", "completed"},
			Pagination: query.Pagination{Limit: 100},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find eligible customer orders: %w", err)
		}
		if len(orders) == 0 {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}

		orderIDs := make([]uuid.UUID, len(orders))
		for i, o := range orders {
			orderIDs[i] = o.ID
		}

		items, err := u.orderItemRepo.ListByOrderIDs(ctx, u.executor, orderIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to list order items: %w", err)
		}

		eligibleOrderIDsMap := make(map[uuid.UUID]bool)
		for _, it := range items {
			if it.ProductID == input.ProductID {
				eligibleOrderIDsMap[it.OrderID] = true
			}
		}
		if len(eligibleOrderIDsMap) == 0 {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}

		reviewedIDs, err := u.reviewRepo.GetReviewedOrderIDs(ctx, u.executor, input.CustomerID, input.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to check reviewed orders: %w", err)
		}
		reviewedMap := make(map[uuid.UUID]bool)
		for _, rID := range reviewedIDs {
			reviewedMap[rID] = true
		}

		var chosenID *uuid.UUID
		for _, o := range orders {
			if eligibleOrderIDsMap[o.ID] && !reviewedMap[o.ID] {
				chosenID = &o.ID
				break
			}
		}

		if chosenID == nil {
			return nil, apperrors.NewConflict(domain.ErrDuplicateReview.Error())
		}

		targetOrderID = *chosenID
	}

	review := &domain.Review{
		ID:         uuid.New(),
		ProductID:  input.ProductID,
		CustomerID: input.CustomerID,
		OrderID:    targetOrderID,
		Rating:     input.Rating,
		Title:      input.Title,
		Comment:    input.Comment,
		CreatedAt:  time.Now(),
	}

	if err := review.Validate(); err != nil {
		return nil, apperrors.NewBadRequest(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(tx transaction.Executor) error {
		if err := u.reviewRepo.Create(ctx, tx, review); err != nil {
			return err
		}

		summary, err := u.reviewRepo.GetRatingSummary(ctx, tx, input.ProductID)
		if err != nil {
			return err
		}

		if err := u.productRepo.UpdateRating(ctx, tx, input.ProductID, summary.AverageRating, summary.ReviewCount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save review and update rating aggregate: %w", err)
	}

	if u.cache != nil {
		_ = u.cache.Delete(ctx, fmt.Sprintf("cache:product:slug:%s", product.Slug))
		_ = u.cache.Delete(ctx, "cache:products:list:all")
	}

	return review, nil
}
