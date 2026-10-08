package reviewusecase_test

import (
	"context"
	"testing"

	"komecore/internal/common/authctx"
	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/review/reviewdomain"
	"komecore/internal/modules/review/reviewusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteReview_CustomerOwner_Success(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	custID := uuid.New()
	reviewID := uuid.New()

	pRepo.products[prodID] = &productdomain.Product{ID: prodID, AverageRating: 5.0, ReviewCount: 1}
	revRepo.reviews[reviewID] = &reviewdomain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: custID,
		Rating:     5,
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, nil, nil, nil, nil, &mockTransactor{})
	actor := &authctx.Actor{
		CustomerID: &custID,
		Type:       authctx.AccountTypeCustomer,
	}

	err := svc.DeleteReview(ctx, reviewusecase.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	require.NoError(t, err)

	assert.True(t, revRepo.deletedReviews[reviewID])
	assert.Equal(t, float64(0.0), pRepo.updatedRatings[prodID])
	assert.Equal(t, 0, pRepo.updatedReviewCount[prodID])
}

func TestDeleteReview_CustomerNotOwner_Forbidden(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	ownerCustID := uuid.New()
	attackerCustID := uuid.New()
	reviewID := uuid.New()

	revRepo.reviews[reviewID] = &reviewdomain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: ownerCustID,
		Rating:     5,
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, nil, nil, nil, nil, &mockTransactor{})
	actor := &authctx.Actor{
		CustomerID: &attackerCustID,
		Type:       authctx.AccountTypeCustomer,
	}

	err := svc.DeleteReview(ctx, reviewusecase.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	require.Error(t, err)
	assert.True(t, apperrors.IsForbidden(err))
}

func TestDeleteReview_StaffAdmin_Success(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	ownerCustID := uuid.New()
	reviewID := uuid.New()

	pRepo.products[prodID] = &productdomain.Product{ID: prodID}
	revRepo.reviews[reviewID] = &reviewdomain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: ownerCustID,
		Rating:     5,
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, nil, nil, nil, nil, &mockTransactor{})
	staffID := uuid.New()
	actor := &authctx.Actor{
		StaffID: &staffID,
		Type:    authctx.AccountTypeStaff,
		Roles:   []authctx.Role{{Code: authctx.RoleStaffAdmin}},
	}

	err := svc.DeleteReview(ctx, reviewusecase.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	require.NoError(t, err)
	assert.True(t, revRepo.deletedReviews[reviewID])
}
