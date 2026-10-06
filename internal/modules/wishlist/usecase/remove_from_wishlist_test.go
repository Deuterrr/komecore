package usecase_test

import (
	"context"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveFromWishlist_Success(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	custID := uuid.New()
	prodID := uuid.New()

	_ = wishlistRepo.Add(ctx, nil, domain.WishlistItem{
		CustomerID: custID,
		ProductID:  prodID,
		CreatedAt:  time.Now(),
	})

	uc := usecase.NewRemoveFromWishlistUsecase(wishlistRepo, nil)
	err := uc.Execute(ctx, usecase.RemoveFromWishlistInput{CustomerID: custID, ProductID: prodID})
	require.NoError(t, err)

	exists, _ := wishlistRepo.Exists(ctx, nil, custID, prodID)
	assert.False(t, exists)
}

func TestRemoveFromWishlist_NotFound(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	uc := usecase.NewRemoveFromWishlistUsecase(wishlistRepo, nil)

	err := uc.Execute(ctx, usecase.RemoveFromWishlistInput{CustomerID: uuid.New(), ProductID: uuid.New()})
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}
