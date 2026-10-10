package productpersistence

import (
	"context"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/productdomain"
	"komecore/internal/modules/product/productrepo"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type ProductPerformanceRepository struct{}

func NewProductPerformanceRepository() *ProductPerformanceRepository {
	return &ProductPerformanceRepository{}
}

func (r *ProductPerformanceRepository) UpsertPerformance(
	ctx context.Context,
	exec transaction.Executor,
	perf productdomain.ProductPerformance,
	basePrice int64,
) error {
	query := `
		INSERT INTO product_performance (
			product_id,
			cost_price,
			supplier_lead_time_days,
			gross_margin_pct,
			created_at
		)
		VALUES (
			$1,
			$2::bigint,
			$3,
			CASE WHEN $2::bigint IS NOT NULL AND $4::bigint > 0
				 THEN ROUND((($4::bigint - $2::bigint)::numeric / $4::bigint) * 100, 2)
				 ELSE NULL
			END,
			NOW()
		)
		ON CONFLICT (product_id) DO UPDATE SET
			cost_price              = EXCLUDED.cost_price,
			supplier_lead_time_days = EXCLUDED.supplier_lead_time_days,
			gross_margin_pct        = EXCLUDED.gross_margin_pct,
			updated_at              = NOW()
	`

	_, err := exec.Exec(ctx, query,
		perf.ProductID,
		perf.CostPrice,
		perf.SupplierLeadTimeDays,
		basePrice,
	)
	if err != nil {
		return fmt.Errorf("upsert product performance failed: %w", err)
	}

	return nil
}

func (r *ProductPerformanceRepository) IncrementViewCount(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
) error {
	query := `
		INSERT INTO product_performance (
			product_id,
			view_count,
			created_at
		)
		VALUES ($1,1,NOW())
		ON CONFLICT (product_id) DO UPDATE SET
			view_count = product_performance.view_count + 1,
			updated_at = NOW()
	`

	_, err := exec.Exec(ctx, query,
		productID,
	)
	if err != nil {
		return fmt.Errorf("increment view count failed: %w", err)
	}

	return nil
}

func (r *ProductPerformanceRepository) GetProductStats(
	ctx context.Context,
	exec transaction.Executor,
	params productrepo.GetProductStatsParams,
) ([]productdomain.ProductStats, int, error) {
	whereClause := ""
	notDeletedCondition := "p.deleted_at IS NULL"

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	conditions = append(conditions, notDeletedCondition)

	if params.ID != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.id = $%d", argPos),
		)
		args = append(args, *params.ID)
		argPos++
	}

	if params.Name != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.name ILIKE $%d", argPos),
		)
		args = append(args, "%"+*params.Name+"%")
		argPos++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM products p
	` + whereClause

	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count product stats failed: %w", err)
	}

	// CTE query for orders and sales
	cteQuery := `
		WITH total_revenue_30d AS (
			SELECT COALESCE(SUM(oi.subtotal), 0) AS revenue
			FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE o.created_at >= NOW() - INTERVAL '30 days'
			  AND o.status != 'cancelled'
		),
		product_sales AS (
			SELECT
				oi.product_id,
				SUM(CASE WHEN o.created_at >= NOW() - INTERVAL '7 days'
						 THEN oi.quantity ELSE 0 END)::int  AS units_7d,
				SUM(CASE WHEN o.created_at >= NOW() - INTERVAL '30 days'
						 THEN oi.quantity ELSE 0 END)::int  AS units_30d,
				SUM(CASE WHEN o.created_at >= NOW() - INTERVAL '90 days'
						 THEN oi.quantity ELSE 0 END)::int  AS units_90d,
				SUM(CASE WHEN o.created_at >= NOW() - INTERVAL '30 days'
						 THEN oi.subtotal ELSE 0 END)       AS revenue_30d
			FROM order_items oi
			JOIN orders o ON o.id = oi.order_id
			WHERE o.created_at >= NOW() - INTERVAL '90 days'
			  AND o.status != 'cancelled'
			GROUP BY oi.product_id
		)
	`

	selectQuery := `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.slug,
			p.description,
			p.status,
			p.base_price,
			p.weight,
			p.created_at,
			p.updated_at,
			p.archived_at,

			pp.cost_price,
			pp.supplier_lead_time_days,
			pp.gross_margin_pct,

			COALESCE(pp.view_count, 0) AS view_count,

			COALESCE(SUM(i.stock), 0)::int AS total_stock,
			COALESCE(SUM(i.reserved_stock), 0)::int AS reserved_stock,

			COALESCE(ps.units_7d,  0) AS sales_velocity_7d,
			COALESCE(ps.units_30d, 0) AS sales_velocity_30d,
			COALESCE(ps.units_90d, 0) AS sales_velocity_90d,

			CASE WHEN COALESCE(pp.view_count, 0) > 0
				 THEN ROUND(COALESCE(ps.units_30d, 0)::numeric / pp.view_count * 100, 2)
				 ELSE 0
			END AS conversion_rate,

			CASE WHEN tr.revenue > 0
				 THEN ROUND(COALESCE(ps.revenue_30d, 0)::numeric / tr.revenue * 100, 2)
				 ELSE 0
			END AS revenue_contrib_pct
	`

	fromQuery := `
		FROM products p
		LEFT JOIN product_performance pp
			ON pp.product_id = p.id
		LEFT JOIN inventory i
			ON i.product_id  = p.id
		LEFT JOIN product_sales ps
			ON ps.product_id = p.id
		CROSS JOIN total_revenue_30d tr
	`

	groupByQuery := `
		GROUP BY
			p.id,
			pp.cost_price,
			pp.supplier_lead_time_days,
			pp.gross_margin_pct,
			pp.view_count,
			ps.units_7d,
			ps.units_30d,
			ps.units_90d,
			ps.revenue_30d,
			tr.revenue
	`

	// Build sorting clauses
	var sortKeys = map[pagination.SortKey]string{
		productrepo.ProductSortLatest:      "p.created_at",
		productrepo.ProductSortName:        "p.name",
		productrepo.ProductSortPrice:       "p.base_price",
		productrepo.ProductSortViewCount:   "COALESCE(pp.view_count, 0)",
		productrepo.ProductSortSales30d:    "COALESCE(ps.units_30d, 0)",
		productrepo.ProductSortSales7d:     "COALESCE(ps.units_7d, 0)",
		productrepo.ProductSortRevenue:     "COALESCE(ps.revenue_30d, 0)",
		productrepo.ProductSortGrossMargin: "COALESCE(pp.gross_margin_pct, 0)",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := sortKeys[sort.By]
		if !exists {
			continue
		}

		direction := "DESC"
		if sort.Direction == pagination.SortAsc {
			direction = "ASC"
		}

		sortClauses = append(
			sortClauses,
			fmt.Sprintf("%s %s", colName, direction),
		)
	}

	orderBy := "ORDER BY p.created_at DESC"
	if len(sortClauses) > 0 {
		orderBy = "ORDER BY " + strings.Join(sortClauses, ", ")
	}

	// Apply pagination
	limitPos := argPos
	offsetPos := argPos + 1

	limit := params.Pagination.Limit
	if limit <= 0 {
		limit = 10
	}

	page := params.Pagination.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	args = append(args, limit, offset)

	fullQuery := cteQuery +
		selectQuery +
		fromQuery +
		whereClause +
		groupByQuery +
		" " +
		orderBy +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPos, offsetPos)

	rows, err := exec.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query product stats failed: %w", err)
	}
	defer rows.Close()

	var results []productdomain.ProductStats
	for rows.Next() {
		var item productdomain.ProductStats
		var costPrice *int64
		var supplierLeadTimeDays *int
		var grossMarginPct *float64

		err := rows.Scan(
			&item.Product.ID,
			&item.Product.SKU,
			&item.Product.Name,
			&item.Product.Slug,
			&item.Product.Description,
			&item.Product.Status,
			&item.Product.Price,
			&item.Product.Weight,
			&item.Product.CreatedAt,
			&item.Product.UpdatedAt,
			&item.Product.ArchivedAt,

			&costPrice,
			&supplierLeadTimeDays,
			&grossMarginPct,
			&item.Performance.ViewCount,

			&item.TotalStock,
			&item.ReservedStock,

			&item.SalesVelocity7d,
			&item.SalesVelocity30d,
			&item.SalesVelocity90d,

			&item.ConversionRate,
			&item.RevenueContribPct,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("mapping product stats failed: %w", err)
		}

		item.Performance.ProductID = item.Product.ID
		item.Performance.CostPrice = costPrice
		item.Performance.SupplierLeadTimeDays = supplierLeadTimeDays
		item.Performance.GrossMarginPct = grossMarginPct

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate product stats failed: %w", err)
	}

	return results, total, nil
}
