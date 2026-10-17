//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/cache"
	"komecore/internal/modules/order"
	"komecore/internal/modules/order/orderpersistence"
	"komecore/internal/modules/product/productpersistence"
	"komecore/internal/modules/review"
	"komecore/internal/modules/review/reviewpersistence"
	"komecore/test/testutil/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Review_VerifiedPurchaseInvariant(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tdb, err := testdb.NewTestDB(ctx)
	skipIfDockerUnavailable(t, err)
	require.NoError(t, err)
	defer func() {
		_ = tdb.Close(ctx)
	}()

	err = tdb.ApplyMigrations()
	require.NoError(t, err)
	defer func() {
		_ = tdb.Truncate(ctx)
	}()

	// 1. Seed Customer, Shop, Product
	_, _, customerID, err := seedUserAndCustomer(ctx, tdb.Pool)
	require.NoError(t, err)

	addressID, err := seedCustomerAddress(ctx, tdb.Pool, customerID)
	require.NoError(t, err)

	shopID, productID, err := seedShopAndProduct(ctx, tdb.Pool, 50)
	require.NoError(t, err)

	// 2. Seed Order with status = pending (unfulfilled)
	orderID := uuid.New()
	now := time.Now().UTC()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO orders (id, customer_id, address_id, number, status, subtotal, shipping_fee, total, created_at, updated_at)
		VALUES ($1, $2, $3, 'ORD-REV-001', 'pending', 150000, 0, 150000, $4, $4);
	`, orderID, customerID, addressID, now)
	require.NoError(t, err)

	orderItemID := uuid.New()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO order_items (id, order_id, shop_id, shop_name, product_id, product_name, quantity, unit_price, subtotal, shipping_fee_total, created_at)
		VALUES ($1, $2, $3, 'Florist Boutique', $4, 'Red Rose Bouquet', 1, 150000, 150000, 0, $5);
	`, orderItemID, orderID, shopID, productID, now)
	require.NoError(t, err)

	// 3. Instantiate ReviewService with real repositories
	reviewRepo := reviewpersistence.NewReviewRepository()
	productRatingUpdater := productpersistence.NewProductRepository()
	orderRepo := orderpersistence.NewOrderRepository()
	orderItemRepo := orderpersistence.NewOrderItemRepository()
	memCache := cache.NewMemoryCache()

	reviewSvc := review.NewService(
		reviewRepo,
		productRatingUpdater,
		orderRepo,
		orderItemRepo,
		memCache,
		tdb.Executor,
		tdb.Transactor,
	)

	title := "Stunning Roses"
	comment := "The roses arrived fresh and beautifully wrapped!"

	// 4. Invariant Check: Review on unfulfilled (pending) order must be rejected
	_, err = reviewSvc.CreateReview(ctx, review.CreateReviewInput{
		CustomerID: customerID,
		ProductID:  productID,
		OrderID:    &orderID,
		Rating:     5,
		Title:      &title,
		Comment:    &comment,
	})
	require.Error(t, err, "review on pending order must fail")
	assert.Contains(t, err.Error(), review.ErrUnverifiedPurchase.Error())

	// 5. Update Order status to delivered
	_, err = tdb.Pool.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3;
	`, order.OrderStatusDelivered, now, orderID)
	require.NoError(t, err)

	// 6. Review submission on delivered order must succeed
	rev, err := reviewSvc.CreateReview(ctx, review.CreateReviewInput{
		CustomerID: customerID,
		ProductID:  productID,
		OrderID:    &orderID,
		Rating:     5,
		Title:      &title,
		Comment:    &comment,
	})
	require.NoError(t, err, "review on delivered order must succeed")
	assert.NotNil(t, rev)
	assert.Equal(t, 5, rev.Rating)
	assert.Equal(t, productID, rev.ProductID)
	assert.Equal(t, customerID, rev.CustomerID)

	// Verify product_reviews record in database
	var reviewCount int
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM product_reviews WHERE product_id = $1;", productID).Scan(&reviewCount)
	require.NoError(t, err)
	assert.Equal(t, 1, reviewCount)

	// Verify product rating is updated
	var avgRating float64
	var totalReviews int
	err = tdb.Pool.QueryRow(ctx, "SELECT average_rating, review_count FROM products WHERE id = $1;", productID).Scan(&avgRating, &totalReviews)
	require.NoError(t, err)
	assert.Equal(t, 5.0, avgRating)
	assert.Equal(t, 1, totalReviews)

	// 7. Duplicate review on the same order must be rejected
	_, err = reviewSvc.CreateReview(ctx, review.CreateReviewInput{
		CustomerID: customerID,
		ProductID:  productID,
		OrderID:    &orderID,
		Rating:     4,
		Title:      &title,
		Comment:    &comment,
	})
	require.Error(t, err, "duplicate review must be rejected")
	assert.Contains(t, err.Error(), review.ErrDuplicateReview.Error())
}
