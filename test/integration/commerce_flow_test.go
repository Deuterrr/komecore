//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"komecore/internal/infra/outbox"
	"komecore/internal/infra/transactor"
	"komecore/test/testutil/testdb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CommerceFlow_OrderCreationWithCouponAndInventory(t *testing.T) {
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

	// 1. Seed Customer, Shop, and Product with stock = 10
	_, _, customerID, err := seedUserAndCustomer(ctx, tdb.Pool)
	require.NoError(t, err)

	shopID, productID, err := seedShopAndProduct(ctx, tdb.Pool, 10)
	require.NoError(t, err)

	// 2. Seed Courier
	courierID := uuid.New()
	now := time.Now().UTC()
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO couriers (id, code, name, service, is_active, created_at, updated_at)
		VALUES ($1, 'JNE', 'JNE Express', 'REG', true, $2, $2);
	`, courierID, now)
	require.NoError(t, err)

	// 3. Seed Coupon: 20% discount with max 50,000 cap, quota = 5
	couponID := uuid.New()
	couponCode := "PROMO20"
	_, err = tdb.Pool.Exec(ctx, `
		INSERT INTO coupons (
			id, code, description, type, discount_value, min_spend, max_discount,
			quota_total, quota_remaining, starts_at, expires_at, is_active, created_at, updated_at
		) VALUES (
			$1, $2, '20% off floral orders', 'percentage', 20, 100000, 50000,
			5, 5, $3, $4, true, $3, $3
		);
	`, couponID, couponCode, now.Add(-1*time.Hour), now.Add(24*time.Hour))
	require.NoError(t, err)

	// 4. Execute Atomic Checkout Flow inside transactor
	// Order: 2 items @ 150,000 = 300,000 subtotal
	// 20% discount = 60,000, capped at max_discount 50,000 -> net total = 250,000
	orderID := uuid.New()
	orderNumber := "ORD-TEST-001"
	qtyToBuy := 2
	itemPrice := int64(150000)
	subtotal := int64(qtyToBuy) * itemPrice
	discountAmount := int64(50000)
	totalAmount := subtotal - discountAmount

	err = tdb.Transactor.WithinTransaction(ctx, func(exec transactor.Executor) error {
		// a. Check and reserve stock atomically
		tag, err := exec.Exec(ctx, `
			UPDATE inventory
			SET stock = stock - $1, updated_at = $2
			WHERE product_id = $3 AND shop_id = $4 AND stock >= $1;
		`, qtyToBuy, now, productID, shopID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("insufficient inventory stock")
		}

		// b. Decrement coupon quota atomically
		cTag, err := exec.Exec(ctx, `
			UPDATE coupons
			SET quota_remaining = quota_remaining - 1, updated_at = $1
			WHERE id = $2 AND quota_remaining > 0;
		`, now, couponID)
		if err != nil {
			return err
		}
		if cTag.RowsAffected() == 0 {
			return errors.New("coupon quota exhausted")
		}

		// c. Persist coupon redemption
		redemptionID := uuid.New()
		_, err = exec.Exec(ctx, `
			INSERT INTO coupon_redemptions (id, coupon_id, customer_id, order_id, discount_applied, redeemed_at)
			VALUES ($1, $2, $3, $4, $5, $6);
		`, redemptionID, couponID, customerID, orderID, discountAmount, now)
		if err != nil {
			return err
		}

		// d. Persist order
		_, err = exec.Exec(ctx, `
			INSERT INTO orders (id, customer_id, number, total, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'pending', $5, $5);
		`, orderID, customerID, orderNumber, totalAmount, now)
		if err != nil {
			return err
		}

		// e. Persist order items
		orderItemID := uuid.New()
		_, err = exec.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, shop_id, quantity, price, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7);
		`, orderItemID, orderID, productID, shopID, qtyToBuy, itemPrice, now)
		if err != nil {
			return err
		}

		// f. Enqueue transactional outbox event
		outboxPayload := outbox.OrderCreatedPayload{
			OrderID:       orderID,
			OrderNumber:   orderNumber,
			CustomerEmail: "buyer@example.com",
			CustomerName:  "Test Buyer",
			Total:         totalAmount,
			Items: []outbox.OrderCreatedItem{
				{
					ProductName: "Red Rose Bouquet",
					Quantity:    qtyToBuy,
					UnitPrice:   itemPrice,
					Subtotal:    int64(qtyToBuy) * itemPrice,
				},
			},
		}
		payloadBytes, err := json.Marshal(outboxPayload)
		if err != nil {
			return err
		}

		outboxID := uuid.New()
		_, err = exec.Exec(ctx, `
			INSERT INTO outbox_events (id, event_type, payload, status, retry_count, max_retries, scheduled_at, created_at)
			VALUES ($1, $2, $3, 'pending', 0, 5, $4, $4);
		`, outboxID, outbox.EventOrderCreated, payloadBytes, now)
		return err
	})
	require.NoError(t, err, "atomic order creation with coupon redemption must succeed")

	// 5. Verify database state post-commit
	var stockRemaining int
	err = tdb.Pool.QueryRow(ctx, "SELECT stock FROM inventory WHERE product_id = $1;", productID).Scan(&stockRemaining)
	require.NoError(t, err)
	assert.Equal(t, 8, stockRemaining, "stock should be decremented from 10 to 8")

	var quotaRemaining int
	err = tdb.Pool.QueryRow(ctx, "SELECT quota_remaining FROM coupons WHERE id = $1;", couponID).Scan(&quotaRemaining)
	require.NoError(t, err)
	assert.Equal(t, 4, quotaRemaining, "coupon quota should be decremented from 5 to 4")

	var redemptionCount int
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM coupon_redemptions WHERE order_id = $1;", orderID).Scan(&redemptionCount)
	require.NoError(t, err)
	assert.Equal(t, 1, redemptionCount, "redemption record must be persisted")

	var persistedTotal int64
	err = tdb.Pool.QueryRow(ctx, "SELECT total FROM orders WHERE id = $1;", orderID).Scan(&persistedTotal)
	require.NoError(t, err)
	assert.Equal(t, totalAmount, persistedTotal, "order total must reflect coupon discount")

	var outboxEventCount int
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE event_type = $1 AND status = 'pending';", outbox.EventOrderCreated).Scan(&outboxEventCount)
	require.NoError(t, err)
	assert.Equal(t, 1, outboxEventCount, "outbox event order.created must be pending in database")

	// 6. Test Rollback: Insufficient Stock
	failedOrderID := uuid.New()
	err = tdb.Transactor.WithinTransaction(ctx, func(exec transactor.Executor) error {
		// Attempt to buy 20 units when only 8 remain
		tag, err := exec.Exec(ctx, `
			UPDATE inventory
			SET stock = stock - 20, updated_at = $1
			WHERE product_id = $2 AND shop_id = $3 AND stock >= 20;
		`, now, productID, shopID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("insufficient inventory stock")
		}
		return nil
	})
	require.Error(t, err, "transaction should fail due to insufficient inventory")

	// Verify rollback integrity
	err = tdb.Pool.QueryRow(ctx, "SELECT stock FROM inventory WHERE product_id = $1;", productID).Scan(&stockRemaining)
	require.NoError(t, err)
	assert.Equal(t, 8, stockRemaining, "stock must remain 8 after rolled back transaction")

	var failedOrderCount int
	err = tdb.Pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE id = $1;", failedOrderID).Scan(&failedOrderCount)
	require.NoError(t, err)
	assert.Equal(t, 0, failedOrderCount, "failed order must not exist in database")
}
