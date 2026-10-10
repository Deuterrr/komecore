package staffpersistence

import (
	"context"
	"fmt"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"
)

type AnalyticsRepository struct{}

func NewAnalyticsRepository() *AnalyticsRepository {
	return &AnalyticsRepository{}
}

func (r *AnalyticsRepository) GetRevenueOverTime(
	ctx context.Context,
	exec transaction.Executor,
	interval string,
	since time.Time,
) ([]staffdomain.RevenueBucket, error) {
	if interval == "" {
		interval = "day"
	}

	query := `
		SELECT
			DATE_TRUNC($1, paid_at) as bucket,
			COALESCE(SUM(amount), 0) as revenue,
			COUNT(*) as count
		FROM payments
		WHERE status = 'paid'
		  AND paid_at >= $2
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	rows, err := exec.Query(ctx, query, interval, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue over time: %w", err)
	}
	if rows == nil {
		return []staffdomain.RevenueBucket{}, nil
	}
	defer rows.Close()

	var buckets []staffdomain.RevenueBucket
	for rows.Next() {
		var b staffdomain.RevenueBucket
		if err := rows.Scan(&b.Bucket, &b.Revenue, &b.Count); err != nil {
			return nil, fmt.Errorf("failed to scan revenue bucket: %w", err)
		}
		buckets = append(buckets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in revenue over time: %w", err)
	}

	if buckets == nil {
		buckets = []staffdomain.RevenueBucket{}
	}

	return buckets, nil
}

func (r *AnalyticsRepository) GetOrderPipeline(
	ctx context.Context,
	exec transaction.Executor,
) ([]staffdomain.OrderStatusCount, error) {
	query := `
		SELECT
			status,
			COUNT(*) as count,
			COALESCE(SUM(total), 0) as total_amount
		FROM orders
		GROUP BY status
		ORDER BY count DESC
	`

	rows, err := exec.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query order pipeline: %w", err)
	}
	if rows == nil {
		return []staffdomain.OrderStatusCount{}, nil
	}
	defer rows.Close()

	var pipeline []staffdomain.OrderStatusCount
	for rows.Next() {
		var p staffdomain.OrderStatusCount
		if err := rows.Scan(&p.Status, &p.Count, &p.TotalAmount); err != nil {
			return nil, fmt.Errorf("failed to scan order status count: %w", err)
		}
		pipeline = append(pipeline, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in order pipeline: %w", err)
	}

	if pipeline == nil {
		pipeline = []staffdomain.OrderStatusCount{}
	}

	return pipeline, nil
}

func (r *AnalyticsRepository) GetTopProducts(
	ctx context.Context,
	exec transaction.Executor,
	limit int,
) ([]staffdomain.TopProduct, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			oi.product_id,
			oi.product_name,
			COALESCE(SUM(oi.quantity), 0) as quantity_sold,
			COALESCE(SUM(oi.subtotal), 0) as total_revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.status IN ('confirmed', 'processing', 'shipped', 'delivered')
		GROUP BY oi.product_id, oi.product_name
		ORDER BY total_revenue DESC, quantity_sold DESC
		LIMIT $1
	`

	rows, err := exec.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top products: %w", err)
	}
	if rows == nil {
		return []staffdomain.TopProduct{}, nil
	}
	defer rows.Close()

	var products []staffdomain.TopProduct
	for rows.Next() {
		var p staffdomain.TopProduct
		if err := rows.Scan(&p.ProductID, &p.ProductName, &p.QuantitySold, &p.TotalRevenue); err != nil {
			return nil, fmt.Errorf("failed to scan top product: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in top products: %w", err)
	}

	if products == nil {
		products = []staffdomain.TopProduct{}
	}

	return products, nil
}
