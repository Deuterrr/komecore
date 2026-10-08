package usecase_test

import (
	"context"
	"testing"

	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/review/domain"
	"komecore/internal/modules/review/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReviews_Success(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productDomain.Product{ID: prodID}

	revRepo.reviews[uuid.New()] = &domain.Review{
		ID:        uuid.New(),
		ProductID: prodID,
		Rating:    5,
	}
	revRepo.reviews[uuid.New()] = &domain.Review{
		ID:        uuid.New(),
		ProductID: prodID,
		Rating:    4,
	}

	svc := usecase.NewReviewService(revRepo, pRepo, nil, nil, nil, nil, nil)
	res, err := svc.ListReviews(ctx, usecase.ListReviewsInput{
		ProductID: prodID,
		Page:      1,
		Limit:     10,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, res.Total)
	assert.Equal(t, 4.5, res.AverageRating)
	assert.Equal(t, 2, res.ReviewCount)
	assert.Len(t, res.Reviews, 2)
}
