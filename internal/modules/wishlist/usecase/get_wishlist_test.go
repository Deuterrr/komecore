package usecase_test

import (
	"context"
	"testing"
	"time"

	inventoryDomain "komecore/internal/modules/inventory/domain"
	productDomain "komecore/internal/modules/product/domain"
	"komecore/internal/modules/wishlist/domain"
	"komecore/internal/modules/wishlist/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWishlist_Success(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	custID := uuid.New()
	prodID := uuid.New()

	_ = wishlistRepo.Add(ctx, nil, domain.WishlistItem{
		CustomerID: custID,
		ProductID:  prodID,
		CreatedAt:  time.Now(),
	})

	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*productDomain.Product{
			prodID: {
				ID:     prodID,
				SKU:    "SNK-001",
				Name:   "Running Shoes",
				Slug:   "running-shoes",
				Status: productDomain.ProductStatusActive,
				Price:  750000,
			},
		},
	}

	invRepo := &mockInventoryRepo{
		inventories: map[uuid.UUID][]inventoryDomain.Inventory{
			prodID: {
				{ProductID: prodID, TotalStock: 15, ReservedStock: 2},
			},
		},
	}

	uc := usecase.NewGetWishlistUsecase(wishlistRepo, pRepo, invRepo, nil, nil, nil)
	items, err := uc.Execute(ctx, custID)
	require.NoError(t, err)
	require.Len(t, items, 1)

	assert.Equal(t, prodID, items[0].ProductID)
	assert.Equal(t, "Running Shoes", items[0].Name)
	assert.Equal(t, int64(750000), items[0].Price)
	assert.True(t, items[0].InStock)
	assert.Equal(t, 15, items[0].TotalStock)
}

func TestGetWishlist_Empty(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	custID := uuid.New()

	uc := usecase.NewGetWishlistUsecase(wishlistRepo, nil, nil, nil, nil, nil)
	items, err := uc.Execute(ctx, custID)
	require.NoError(t, err)
	assert.Empty(t, items)
}
