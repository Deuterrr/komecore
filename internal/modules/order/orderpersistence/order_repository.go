package orderpersistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/pagination"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*orderdomain.Order, error) {
	query := `
		SELECT
			id,
			number,
			customer_id,
			address_id,
			status,
			subtotal,
			shipping_fee,
			total,
			confirmed_at,
			handling_expires_at,
			created_at,
			updated_at
		FROM
			orders
		WHERE
			id = $1
		LIMIT 1
	`

	var order orderdomain.Order
	err := exec.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.Number,
		&order.CustomerID,
		&order.AddressID,
		&order.Status,
		&order.Subtotal,
		&order.ShippingFee,
		&order.Total,
		&order.ConfirmedAt,
		&order.HandlingExpiresAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query order by id failed: %w", err)
	}

	return &order, nil
}

func (r *OrderRepository) GetByNumber(
	ctx context.Context,
	exec transaction.Executor,
	number string,
) (*orderdomain.Order, error) {
	query := `
		SELECT
			id,
			number,
			customer_id,
			address_id,
			status,
			subtotal,
			shipping_fee,
			total,
			confirmed_at,
			handling_expires_at,
			created_at,
			updated_at
		FROM
			orders
		WHERE
			number = $1
		LIMIT 1
	`

	var order orderdomain.Order
	err := exec.QueryRow(ctx, query, number).Scan(
		&order.ID,
		&order.Number,
		&order.CustomerID,
		&order.AddressID,
		&order.Status,
		&order.Subtotal,
		&order.ShippingFee,
		&order.Total,
		&order.ConfirmedAt,
		&order.HandlingExpiresAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query order by number failed: %w", err)
	}

	return &order, nil
}

func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	status orderdomain.OrderStatus,
) error {
	query := `
		UPDATE orders
		SET
			status = $2
		WHERE
			id = $1
	`

	_, err := exec.Exec(ctx, query,
		id,
		status,
	)
	if err != nil {
		return fmt.Errorf("query to update status: %w", err)
	}

	return nil
}

func (r *OrderRepository) UpdateStatusWithSLA(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	status orderdomain.OrderStatus,
	confirmedAt *time.Time,
	expiresAt *time.Time,
) error {
	query := `
		UPDATE orders
		SET
			status              = $2,
			confirmed_at        = COALESCE($3, confirmed_at),
			handling_expires_at = COALESCE($4, handling_expires_at),
			updated_at          = NOW()
		WHERE
			id = $1
	`

	_, err := exec.Exec(ctx, query,
		id,
		status,
		confirmedAt,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("query to update status with SLA: %w", err)
	}

	return nil
}

func (r *OrderRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	order orderdomain.Order,
) error {
	query := `
		INSERT INTO orders (
			id,
			number,
			customer_id,
			address_id,
			status,
			subtotal,
			shipping_fee,
			total,
			confirmed_at,
			handling_expires_at,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id)
		DO UPDATE SET
			number = EXCLUDED.number,
			customer_id = EXCLUDED.customer_id,
			address_id = EXCLUDED.address_id,
			status = EXCLUDED.status,
			subtotal = EXCLUDED.subtotal,
			shipping_fee = EXCLUDED.shipping_fee,
			total = EXCLUDED.total,
			confirmed_at = EXCLUDED.confirmed_at,
			handling_expires_at = EXCLUDED.handling_expires_at
	`

	_, err := exec.Exec(ctx, query,
		order.ID,
		order.Number,
		order.CustomerID,
		order.AddressID,
		order.Status,
		order.Subtotal,
		order.ShippingFee,
		order.Total,
		order.ConfirmedAt,
		order.HandlingExpiresAt,
		order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("query to save order: %w", err)
	}

	return nil
}

func (r *OrderRepository) FindOrders(
	ctx context.Context,
	exec transaction.Executor,
	params orderrepo.FindOrderParams,
) ([]orderdomain.Order, int, error) {
	baseQuery := `
		FROM orders o
	`

	selectQuery := `
		SELECT
			o.id,
			o.number,
			o.customer_id,
			o.address_id,
			o.status,
			o.subtotal,
			o.shipping_fee,
			o.total,
			o.confirmed_at,
			o.handling_expires_at,
			o.created_at,
			o.updated_at
	`

	whereClause := ""
	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	if params.ID != nil {
		conditions = append(conditions, fmt.Sprintf("o.id = $%d", argPos))
		args = append(args, *params.ID)
		argPos++
	}

	if params.Number != nil {
		conditions = append(conditions, fmt.Sprintf("o.number ILIKE $%d", argPos))
		args = append(args, "%"+*params.Number+"%")
		argPos++
	}

	if params.CustomerID != nil {
		conditions = append(conditions, fmt.Sprintf("o.customer_id = $%d", argPos))
		args = append(args, *params.CustomerID)
		argPos++
	}

	if len(params.ShopIDs) > 0 {
		shopIDStrings := make([]string, len(params.ShopIDs))
		for i, id := range params.ShopIDs {
			shopIDStrings[i] = id.String()
		}
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = o.id AND oi.shop_id = ANY($%d::uuid[]))", argPos))
		args = append(args, shopIDStrings)
		argPos++
	} else if params.ShopID != nil {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM order_items oi WHERE oi.order_id = o.id AND oi.shop_id = $%d)", argPos))
		args = append(args, *params.ShopID)
		argPos++
	}

	if len(params.Statuses) > 0 {
		placeholders := make([]string, len(params.Statuses))
		for i, s := range params.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, s)
			argPos++
		}
		conditions = append(conditions, fmt.Sprintf("o.status IN (%s)", strings.Join(placeholders, ", ")))
	} else if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argPos))
		args = append(args, *params.Status)
		argPos++
	}

	if params.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("o.created_at >= $%d", argPos))
		args = append(args, *params.FromDate)
		argPos++
	}

	if params.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("o.created_at <= $%d", argPos))
		args = append(args, *params.ToDate)
		argPos++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count matching orders
	countQuery := `
		SELECT COUNT(DISTINCT o.id)
	` + baseQuery + whereClause

	countArgs := append([]any{}, args...)
	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count orders failed: %w", err)
	}

	// Build sorting expressions
	var orderSortKeys = map[pagination.SortKey]string{
		orderrepo.OrderSortLatest: "o.created_at",
		orderrepo.OrderSortNumber: "o.number",
		orderrepo.OrderSortTotal:  "o.total",
		orderrepo.OrderSortStatus: "o.status",
		orderrepo.OrderSortModify: "o.updated_at",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := orderSortKeys[sort.By]
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

	orderBy := "ORDER BY o.created_at DESC"
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

	// The execution
	queryStr := selectQuery +
		baseQuery +
		whereClause +
		" " +
		orderBy +
		fmt.Sprintf(
			" LIMIT $%d OFFSET $%d",
			limitPos,
			offsetPos,
		)

	rows, err := exec.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query orders failed: %w", err)
	}
	defer rows.Close()

	var results []orderdomain.Order
	for rows.Next() {
		var item orderdomain.Order
		err := rows.Scan(
			&item.ID,
			&item.Number,
			&item.CustomerID,
			&item.AddressID,
			&item.Status,
			&item.Subtotal,
			&item.ShippingFee,
			&item.Total,
			&item.ConfirmedAt,
			&item.HandlingExpiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("mapping order model to domain failed: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders failed: %w", err)
	}

	return results, total, nil
}

func (r *OrderRepository) SetConfirmedAndExpiry(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
	confirmedAt time.Time,
	expiresAt time.Time,
) error {
	query := `
		UPDATE orders
		SET
			confirmed_at        = $2,
			handling_expires_at = $3,
			updated_at          = NOW()
		WHERE
			id = $1
	`

	_, err := exec.Exec(ctx, query, id, confirmedAt, expiresAt)
	if err != nil {
		return fmt.Errorf("query to set confirmed and expiry failed: %w", err)
	}

	return nil
}

func (r *OrderRepository) FindExpiredUnfulfilledOrders(
	ctx context.Context,
	exec transaction.Executor,
	now time.Time,
	limit int,
) ([]orderdomain.Order, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id,
			number,
			customer_id,
			address_id,
			status,
			subtotal,
			shipping_fee,
			total,
			confirmed_at,
			handling_expires_at,
			created_at,
			updated_at
		FROM
			orders
		WHERE
			status IN ('confirmed', 'processing')
			AND handling_expires_at IS NOT NULL
			AND handling_expires_at <= $1
		ORDER BY handling_expires_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	rows, err := exec.Query(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("query expired unfulfilled orders failed: %w", err)
	}
	defer rows.Close()

	var results []orderdomain.Order
	for rows.Next() {
		var item orderdomain.Order
		err := rows.Scan(
			&item.ID,
			&item.Number,
			&item.CustomerID,
			&item.AddressID,
			&item.Status,
			&item.Subtotal,
			&item.ShippingFee,
			&item.Total,
			&item.ConfirmedAt,
			&item.HandlingExpiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan expired unfulfilled order failed: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired unfulfilled orders failed: %w", err)
	}

	return results, nil
}
