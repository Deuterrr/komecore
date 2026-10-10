package reviewusecase_test

import (
	"context"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/review/reviewusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReview_SuccessWithDeliveredOrder(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productdomain.Product{ID: prodID, Name: "Book", Slug: "book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderdomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderdomain.OrderStatusDelivered},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderdomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	comment := "Great read!"
	title := "Loved it"
	review, err := svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
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
	svc := reviewusecase.NewReviewService(nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		Rating:     0,
	})
	assert.True(t, apperror.IsBadRequest(err))

	_, err = svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
		Rating:     6,
	})
	assert.True(t, apperror.IsBadRequest(err))
}

func TestCreateReview_UnverifiedPurchase_NoOrders(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productdomain.Product{ID: prodID, Name: "Book"}

	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderdomain.OrderItem{}}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	_, err := svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: uuid.New(),
		ProductID:  prodID,
		Rating:     5,
	})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
	assert.Contains(t, err.Error(), "Customer has not purchased this product")
}

func TestCreateReview_UnverifiedPurchase_OrderNotDelivered(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productdomain.Product{ID: prodID, Name: "Book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderdomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderdomain.OrderStatusShipped},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderdomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})
	_, err := svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		OrderID:    &orderID,
		Rating:     4,
	})

	require.Error(t, err)
	assert.True(t, apperror.IsForbidden(err))
}

func TestCreateReview_DuplicateReviewPrevented(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productdomain.Product{ID: prodID, Name: "Book"}

	custID := uuid.New()
	orderID := uuid.New()

	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderdomain.Order{
			orderID: {ID: orderID, CustomerID: custID, Status: orderdomain.OrderStatusDelivered},
		},
	}
	oiRepo := &mockOrderItemRepo{
		items: map[uuid.UUID][]orderdomain.OrderItem{
			orderID: {{ID: uuid.New(), OrderID: orderID, ProductID: prodID}},
		},
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, oRepo, oiRepo, nil, nil, &mockTransactor{})

	// 1st review succeeds
	_, err := svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		Rating:     5,
	})
	require.NoError(t, err)

	// 2nd review for same purchase fails with Conflict
	_, err = svc.CreateReview(ctx, reviewusecase.CreateReviewInput{
		CustomerID: custID,
		ProductID:  prodID,
		Rating:     4,
	})
	require.Error(t, err)
	assert.True(t, apperror.IsConflict(err))
}
