package usecase_test

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	orderDomain "komecore/internal/modules/order/domain"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/review/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReview_SuccessWithDeliveredOrder(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productDomain.Product{ID: prodID, Name: "Book", Slug: "book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderDomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderDomain.OrderStatusDelivered},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderDomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := usecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	comment := "Great read!"
	title := "Loved it"
	review, err := svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		Rating:     5,
		Title:      &title,
		Comment:    &comment,
	})

	require.NoError(t, err)
	require.NotNil(t, review)
	assert.Equal(t, 5, review.Rating)
	assert.Equal(t, orderID, review.OrderID)
	assert.Equal(t, float64(5.0), pRepo.updatedRatings[prodID])
	assert.Equal(t, 1, pRepo.updatedReviewCount[prodID])
}

func TestCreateReview_InvalidRating(t *testing.T) {
	ctx := context.Background()
	svc := usecase.NewReviewService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		Rating:     0,
	})
	assert.True(t, apperrors.IsBadRequest(err))

	_, err = svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		Rating:     6,
	})
	assert.True(t, apperrors.IsBadRequest(err))
}

func TestCreateReview_UnverifiedPurchase_NoOrders(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productDomain.Product{ID: prodID, Name: "Book"}

	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}

	svc := usecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	_, err := svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  prodID,
		Rating:     5,
	})

	require.Error(t, err)
	assert.True(t, apperrors.IsForbidden(err))
	assert.Contains(t, err.Error(), "Customer has not purchased this product")
}

func TestCreateReview_UnverifiedPurchase_OrderNotDelivered(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productDomain.Product{ID: prodID, Name: "Book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderDomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderDomain.OrderStatusShipped},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderDomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := usecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	_, err := svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		OrderID:    &orderID,
		Rating:     4,
	})

	require.Error(t, err)
	assert.True(t, apperrors.IsForbidden(err))
}

func TestCreateReview_DuplicateReviewPrevented(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productDomain.Product{ID: prodID, Name: "Book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderDomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderDomain.OrderStatusDelivered},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderDomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := usecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})

	// 1st review succeeds
	_, err := svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		Rating:     5,
	})
	require.NoError(t, err)

	// 2nd review for same purchase fails with Conflict
	_, err = svc.CreateReview(ctx, usecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		Rating:     4,
	})
	require.Error(t, err)
	assert.True(t, apperrors.IsConflict(err))
}
