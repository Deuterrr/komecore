package wishlist_test

import (
	"context"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/product"
	"komecore/internal/modules/wishlist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToWishlist_Success(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	prodID := uuid.New()
	custID := uuid.New()

	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*product.Product{
			prodID: {ID: prodID, Name: "Sneakers", Status: product.ProductStatusActive, Price: 500000},
		},
	}

	svc := wishlist.NewService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{
		CustomerID: custID,
		ProductID:  prodID,
	})

	require.NoError(t, err)
	exists, _ := wishlistRepo.Exists(ctx, nil, custID, prodID)
	assert.True(t, exists)
}

func TestAddToWishlist_DuplicateRejected(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	prodID := uuid.New()
	custID := uuid.New()

	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*product.Product{
			prodID: {ID: prodID, Name: "Sneakers"},
		},
	}

	svc := wishlist.NewService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.NoError(t, err)

	// Second attempt should fail with conflict
	err = svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.Error(t, err)
	assert.True(t, apperror.IsConflict(err))
}

func TestAddToWishlist_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*product.Product{},
	}

	svc := wishlist.NewService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestAddToWishlist_InvalidInput(t *testing.T) {
	ctx := context.Background()
	svc := wishlist.NewService(nil, nil, nil, nil, nil, nil)

	err := svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{CustomerID: uuid.Nil, ProductID: uuid.New()})
	require.Error(t, err)

	err = svc.AddToWishlist(ctx, wishlist.AddToWishlistInput{CustomerID: uuid.New(), ProductID: uuid.Nil})
	require.Error(t, err)
}
