package usecase_test

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	authDomain "komecore/internal/modules/auth/domain"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/usecase"

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

	pRepo.products[prodID] = &productDomain.Product{ID: prodID, AverageRating: 5.0, ReviewCount: 1}
	revRepo.reviews[reviewID] = &domain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: custID,
		Rating:     5,
	}

	uc := usecase.NewDeleteReviewUsecase(revRepo, pRepo, nil, nil, &mockTransactor{})
	actor := &authDomain.Actor{
		CustomerID: &custID,
		Type:       authDomain.AccountTypeCustomer,
	}

	err := uc.Execute(ctx, usecase.DeleteReviewInput{
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

	revRepo.reviews[reviewID] = &domain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: ownerCustID,
		Rating:     5,
	}

	uc := usecase.NewDeleteReviewUsecase(revRepo, pRepo, nil, nil, &mockTransactor{})
	actor := &authDomain.Actor{
		CustomerID: &attackerCustID,
		Type:       authDomain.AccountTypeCustomer,
	}

	err := uc.Execute(ctx, usecase.DeleteReviewInput{
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

	pRepo.products[prodID] = &productDomain.Product{ID: prodID}
	revRepo.reviews[reviewID] = &domain.Review{
		ID:         reviewID,
		ProductID:  prodID,
		CustomerID: ownerCustID,
		Rating:     5,
	}

	uc := usecase.NewDeleteReviewUsecase(revRepo, pRepo, nil, nil, &mockTransactor{})
	staffID := uuid.New()
	actor := &authDomain.Actor{
		StaffID: &staffID,
		Type:    authDomain.AccountTypeStaff,
		Roles:   []authDomain.Role{{Code: authDomain.RoleStaffAdmin}},
	}

	err := uc.Execute(ctx, usecase.DeleteReviewInput{
		ReviewID: reviewID,
		Actor:    actor,
	})
	require.NoError(t, err)
	assert.True(t, revRepo.deletedReviews[reviewID])
}
