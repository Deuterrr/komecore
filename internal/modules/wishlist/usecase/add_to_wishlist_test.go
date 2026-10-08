package usecase_test

import (
	"context"
	"testing"

	apperrors "komecore/internal/common/errors"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/wishlist/usecase"

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
		products: map[uuid.UUID]*productDomain.Product{
			prodID: {ID: prodID, Name: "Sneakers", Status: productDomain.ProductStatusActive, Price: 500000},
		},
	}

	svc := usecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, usecase.AddToWishlistInput{
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
		products: map[uuid.UUID]*productDomain.Product{
			prodID: {ID: prodID, Name: "Sneakers"},
		},
	}

	svc := usecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, usecase.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.NoError(t, err)

	// Second attempt should fail with conflict
	err = svc.AddToWishlist(ctx, usecase.AddToWishlistInput{CustomerID: custID, ProductID: prodID})
	require.Error(t, err)
	assert.True(t, apperrors.IsConflict(err))
}

func TestAddToWishlist_ProductNotFound(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*productDomain.Product{},
	}

	svc := usecase.NewWishlistService(wishlistRepo, pRepo, nil, nil, nil, nil)
	err := svc.AddToWishlist(ctx, usecase.AddToWishlistInput{
		CustomerID: uuid.New(),
		ProductID:  uuid.New(),
	})
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestAddToWishlist_InvalidInput(t *testing.T) {
	ctx := context.Background()
	svc := usecase.NewWishlistService(nil, nil, nil, nil, nil, nil)

	err := svc.AddToWishlist(ctx, usecase.AddToWishlistInput{CustomerID: uuid.Nil, ProductID: uuid.New()})
	require.Error(t, err)

	err = svc.AddToWishlist(ctx, usecase.AddToWishlistInput{CustomerID: uuid.New(), ProductID: uuid.Nil})
	require.Error(t, err)
}
