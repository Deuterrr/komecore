package wishlistusecase_test

import (
	"context"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	"komecore/internal/modules/wishlist/wishlistdomain"
	"komecore/internal/modules/wishlist/wishlistusecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveFromWishlist_Success(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	custID := uuid.New()
	prodID := uuid.New()

	_ = wishlistRepo.Add(ctx, nil, wishlistdomain.WishlistItem{
		CustomerID: custID,
		ProductID:  prodID,
		CreatedAt:  time.Now(),
	})

	svc := wishlistusecase.NewWishlistService(wishlistRepo, nil, nil, nil, nil, nil)
	err := svc.RemoveFromWishlist(ctx, wishlistusecase.RemoveFromWishlistInput{CustomerID: custID, ProductID: prodID})
	require.NoError(t, err)

	exists, _ := wishlistRepo.Exists(ctx, nil, custID, prodID)
	assert.False(t, exists)
}

func TestRemoveFromWishlist_NotFound(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	svc := wishlistusecase.NewWishlistService(wishlistRepo, nil, nil, nil, nil, nil)

	err := svc.RemoveFromWishlist(ctx, wishlistusecase.RemoveFromWishlistInput{CustomerID: uuid.New(), ProductID: uuid.New()})
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}
