package reviewusecase_test

import (
	"context"
	"testing"

	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/review/reviewdomain"
	"komecore/internal/modules/review/reviewusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListReviews_Success(t *testing.T) {
	ctx := context.Background()
	revRepo := newMockReviewRepo()
	pRepo := newMockProductRepo()
	prodID := uuid.New()
	pRepo.products[prodID] = &productdomain.Product{ID: prodID}

	revRepo.reviews[uuid.New()] = &reviewdomain.Review{
		ID:        uuid.New(),
		ProductID: prodID,
		Rating:    5,
	}
	revRepo.reviews[uuid.New()] = &reviewdomain.Review{
		ID:        uuid.New(),
		ProductID: prodID,
		Rating:    4,
	}

	svc := reviewusecase.NewReviewService(revRepo, pRepo, nil, nil, nil, nil, nil)
	res, err := svc.ListReviews(ctx, reviewusecase.ListReviewsInput{
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
