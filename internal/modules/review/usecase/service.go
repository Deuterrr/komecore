package usecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	"komecore/internal/infra/cache"
	transaction "komecore/internal/infra/transactor"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
)

type ReviewRepository interface {
	Create(ctx context.Context, exec transaction.Executor, review *domain.Review) error
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*domain.Review, error)
	Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	ListByProductID(ctx context.Context, exec transaction.Executor, params repository.ListReviewsParams) ([]domain.ReviewWithCustomer, int, error)
	GetRatingSummary(ctx context.Context, exec transaction.Executor, productID uuid.UUID) (*domain.ProductRatingSummary, error)
	HasReviewedOrder(ctx context.Context, exec transaction.Executor, customerID, productID, orderID uuid.UUID) (bool, error)
	GetReviewedOrderIDs(ctx context.Context, exec transaction.Executor, customerID, productID uuid.UUID) ([]uuid.UUID, error)
}

type ProductRatingUpdater interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*productDomain.Product, error)
	UpdateRating(ctx context.Context, exec transaction.Executor, id uuid.UUID, averageRating float64, reviewCount int) error
}

type OrderReader interface {
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*orderDomain.Order, error)
	FindOrders(ctx context.Context, exec transaction.Executor, params orderRepo.FindOrderParams) ([]orderDomain.Order, int, error)
}

type OrderItemReader interface {
	ListByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]orderDomain.OrderItem, error)
	ListByOrderIDs(ctx context.Context, exec transaction.Executor, orderIDs []uuid.UUID) ([]orderDomain.OrderItem, error)
}

type CreateReviewInput struct {
	CustomerID uuid.UUID
	ProductID  uuid.UUID
	OrderID    *uuid.UUID
	Rating     int
	Title      *string
	Comment    *string
}

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

type DeleteReviewInput struct {
	ReviewID uuid.UUID
	Actor    *authctx.Actor
}

type ReviewService struct {
	reviewRepo    ReviewRepository
	productRepo   ProductRatingUpdater
	orderRepo     OrderReader
	orderItemRepo OrderItemReader
	cache         cache.Cache
	executor      transaction.Executor
	transactor    transaction.Transactor
}

func NewReviewService(
	reviewRepo ReviewRepository,
	productRepo ProductRatingUpdater,
	orderRepo OrderReader,
	orderItemRepo OrderItemReader,
	cache cache.Cache,
	executor transaction.Executor,
	transactor transaction.Transactor,
) *ReviewService {
	return &ReviewService{
		reviewRepo:    reviewRepo,
		productRepo:   productRepo,
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		cache:         cache,
		executor:      executor,
		transactor:    transactor,
	}
}

func (s *ReviewService) CreateReview(ctx context.Context, input CreateReviewInput) (*domain.Review, error) {
	if input.Rating < 1 || input.Rating > 5 {
		return nil, apperrors.NewBadRequest(domain.ErrInvalidRating.Error())
	}
	if input.CustomerID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invalid customer id")
	}
	if input.ProductID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invalid product id")
	}

	product, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve product: %w", err)
	}
	if product == nil {
		return nil, apperrors.NewNotFound("product not found")
	}

	var targetOrderID uuid.UUID

	if input.OrderID != nil {
		order, err := s.orderRepo.GetByID(ctx, s.executor, *input.OrderID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve order: %w", err)
		}
		if order == nil || order.CustomerID != input.CustomerID {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}
		if order.Status != orderDomain.OrderStatusDelivered && string(order.Status) != "completed" {
			return nil, apperrors.NewForbidden(domain.ErrUnverifiedPurchase.Error())
		}

		items, err := s.orderItemRepo.ListByOrderID(ctx, s.executor, order.ID)
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

		hasReviewed, err := s.reviewRepo.HasReviewedOrder(ctx, s.executor, input.CustomerID, input.ProductID, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check order review status: %w", err)
		}
		if hasReviewed {
			return nil, apperrors.NewConflict(domain.ErrDuplicateReview.Error())
		}

		targetOrderID = order.ID
	} else {
		orders, _, err := s.orderRepo.FindOrders(ctx, s.executor, orderRepo.FindOrderParams{
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

		items, err := s.orderItemRepo.ListByOrderIDs(ctx, s.executor, orderIDs)
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

		reviewedIDs, err := s.reviewRepo.GetReviewedOrderIDs(ctx, s.executor, input.CustomerID, input.ProductID)
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

	err = s.transactor.WithinTransaction(ctx, func(tx transaction.Executor) error {
		if err := s.reviewRepo.Create(ctx, tx, review); err != nil {
			return err
		}

		summary, err := s.reviewRepo.GetRatingSummary(ctx, tx, input.ProductID)
		if err != nil {
			return err
		}

		if err := s.productRepo.UpdateRating(ctx, tx, input.ProductID, summary.AverageRating, summary.ReviewCount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save review and update rating aggregate: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, fmt.Sprintf("cache:product:slug:%s", product.Slug))
		_ = s.cache.Delete(ctx, "cache:products:list:all")
	}

	return review, nil
}

func (s *ReviewService) ListReviews(ctx context.Context, input ListReviewsInput) (*ListReviewsResult, error) {
	if input.ProductID == uuid.Nil {
		return nil, apperrors.NewBadRequest("invalid product id")
	}

	product, err := s.productRepo.GetByID(ctx, s.executor, input.ProductID)
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

	reviews, total, err := s.reviewRepo.ListByProductID(ctx, s.executor, repository.ListReviewsParams{
		ProductID: input.ProductID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list product reviews: %w", err)
	}

	summary, err := s.reviewRepo.GetRatingSummary(ctx, s.executor, input.ProductID)
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

func (s *ReviewService) DeleteReview(ctx context.Context, input DeleteReviewInput) error {
	if input.ReviewID == uuid.Nil {
		return apperrors.NewBadRequest("invalid review id")
	}

	review, err := s.reviewRepo.GetByID(ctx, s.executor, input.ReviewID)
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

	err = s.transactor.WithinTransaction(ctx, func(tx transaction.Executor) error {
		if err := s.reviewRepo.Delete(ctx, tx, input.ReviewID); err != nil {
			return err
		}

		summary, err := s.reviewRepo.GetRatingSummary(ctx, tx, review.ProductID)
		if err != nil {
			return err
		}

		if err := s.productRepo.UpdateRating(ctx, tx, review.ProductID, summary.AverageRating, summary.ReviewCount); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete review and update rating aggregate: %w", err)
	}

	if s.cache != nil {
		product, _ := s.productRepo.GetByID(ctx, s.executor, review.ProductID)
		if product != nil {
			_ = s.cache.Delete(ctx, fmt.Sprintf("cache:product:slug:%s", product.Slug))
		}
		_ = s.cache.Delete(ctx, "cache:products:list:all")
	}

	return nil
}
