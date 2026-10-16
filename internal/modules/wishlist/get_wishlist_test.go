package wishlist_test

import (
	"context"
	"testing"
	"time"

	"komecore/internal/modules/inventory"
	"komecore/internal/modules/product"
	"komecore/internal/modules/wishlist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWishlist_Success(t *testing.T) {
	ctx := context.Background()
	wishlistRepo := newMockWishlistRepo()
	custID := uuid.New()
	prodID := uuid.New()

	_ = wishlistRepo.Add(ctx, nil, wishlist.WishlistItem{
		CustomerID: custID,
		ProductID:  prodID,
		CreatedAt:  time.Now(),
	})

	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*product.Product{
			prodID: {
				ID:     prodID,
				SKU:    "SNK-001",
				Name:   "Running Shoes",
				Slug:   "running-shoes",
				Status: product.ProductStatusActive,
				Price:  750000,
			},
		},
	}

	invRepo := &mockInventoryRepo{
		inventories: map[uuid.UUID][]inventory.Inventory{
			prodID: {
				{ProductID: prodID, TotalStock: 15, ReservedStock: 2},
			},
		},
	}

	svc := wishlist.NewService(wishlistRepo, pRepo, invRepo, nil, nil, nil)
	items, err := svc.GetWishlist(ctx, custID)
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

	svc := wishlist.NewService(wishlistRepo, nil, nil, nil, nil, nil)
	items, err := svc.GetWishlist(ctx, custID)
	require.NoError(t, err)
	assert.Empty(t, items)
}
