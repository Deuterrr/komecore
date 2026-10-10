package wishlistusecase_test

import (
	"context"
	"testing"

	"komecore/internal/apperror"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/wishlist/wishlistusecase"

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
		products: map[uuid.UUID]*productdomain.Product{
			prodID: {ID: prodID, Name: "Sneakers", Status: productdomain.ProductStatusActive, Price: 500000},
		},
	}

	svc := wishlistusecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{
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
		products: map[uuid.UUID]*productdomain.Product{
			prodID: {ID: prodID, Name: "Sneakers"},
		},
	}

	svc := wishlistusecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.NoError(t, err)

	// Second attempt should fail with conflict
	err = svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.Error(t, err)
	assert.True(t, apperror.IsConflict(err))
}

func TestAddToWishlist_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*productdomain.Product{},
	}

	svc := wishlistusecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, apperror.IsNotFound(err))
}

func TestAddToWishlist_InvalidInput(t *testing.T) {
	ctx := context.Background()
	svc := wishlistusecase.NewWishlistService(nil, nil, nil, nil, nil, nil)

	err := svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{CustomerID: uuid.Nil, ProductID: uuid.New()})
	require.Error(t, err)

	err = svc.AddToWishlist(ctx, wishlistusecase.AddToWishlistInput{CustomerID: uuid.New(), ProductID: uuid.Nil})
	require.Error(t, err)
}
