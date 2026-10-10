package discountpersistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/discount/discountdomain"
	"komecore/internal/modules/discount/discountrepo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CouponRepository struct{}

func NewCouponRepository() *CouponRepository {
	return &CouponRepository{}
}

func (r *CouponRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*discountdomain.Coupon, error) {
	query := `
		SELECT
			id, code, description, type, discount_value, min_spend, max_discount,
			quota_total, quota_remaining, starts_at, expires_at, is_active,
			created_at, updated_at
		FROM coupons
		WHERE id = $1
		LIMIT 1
	`

	var coupon discountdomain.Coupon
	err := exec.QueryRow(ctx, query, id).Scan(
		&coupon.ID,
		&coupon.Code,
		&coupon.Description,
		&coupon.Type,
		&coupon.DiscountValue,
		&coupon.MinSpend,
		&coupon.MaxDiscount,
		&coupon.QuotaTotal,
		&coupon.QuotaRemaining,
		&coupon.StartsAt,
		&coupon.ExpiresAt,
		&coupon.IsActive,
		&coupon.CreatedAt,
		&coupon.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, transaction.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get coupon by id failed: %w", err)
	}

	return &coupon, nil
}

func (r *CouponRepository) GetByCode(
	ctx context.Context,
	exec transaction.Executor,
	code string,
) (*discountdomain.Coupon, error) {
	query := `
		SELECT
			id, code, description, type, discount_value, min_spend, max_discount,
			quota_total, quota_remaining, starts_at, expires_at, is_active,
			created_at, updated_at
		FROM coupons
		WHERE UPPER(code) = UPPER($1)
		LIMIT 1
	`

	var coupon discountdomain.Coupon
	err := exec.QueryRow(ctx, query, strings.TrimSpace(code)).Scan(
		&coupon.ID,
		&coupon.Code,
		&coupon.Description,
		&coupon.Type,
		&coupon.DiscountValue,
		&coupon.MinSpend,
		&coupon.MaxDiscount,
		&coupon.QuotaTotal,
		&coupon.QuotaRemaining,
		&coupon.StartsAt,
		&coupon.ExpiresAt,
		&coupon.IsActive,
		&coupon.CreatedAt,
		&coupon.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, transaction.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get coupon by code failed: %w", err)
	}

	return &coupon, nil
}

func (r *CouponRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	coupon discountdomain.Coupon,
) error {
	query := `
		INSERT INTO coupons (
			id, code, description, type, discount_value, min_spend, max_discount,
			quota_total, quota_remaining, starts_at, expires_at, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			code = EXCLUDED.code,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			discount_value = EXCLUDED.discount_value,
			min_spend = EXCLUDED.min_spend,
			max_discount = EXCLUDED.max_discount,
			quota_total = EXCLUDED.quota_total,
			quota_remaining = EXCLUDED.quota_remaining,
			starts_at = EXCLUDED.starts_at,
			expires_at = EXCLUDED.expires_at,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`

	_, err := exec.Exec(
		ctx,
		query,
		coupon.ID,
		discountdomain.NormalizeCode(coupon.Code),
		coupon.Description,
		coupon.Type,
		coupon.DiscountValue,
		coupon.MinSpend,
		coupon.MaxDiscount,
		coupon.QuotaTotal,
		coupon.QuotaRemaining,
		coupon.StartsAt,
		coupon.ExpiresAt,
		coupon.IsActive,
		coupon.CreatedAt,
		coupon.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save coupon failed: %w", err)
	}

	return nil
}

func (r *CouponRepository) List(
	ctx context.Context,
	exec transaction.Executor,
	params discountrepo.ListCouponsParams,
) ([]discountdomain.Coupon, int, error) {
	baseQuery := " FROM coupons c"
	whereClause := ""

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	if params.Code != nil && strings.TrimSpace(*params.Code) != "" {
		conditions = append(conditions, fmt.Sprintf("c.code ILIKE $%d", argPos))
		args = append(args, "%"+strings.TrimSpace(*params.Code)+"%")
		argPos++
	}

	if params.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("c.is_active = $%d", argPos))
		args = append(args, *params.IsActive)
		argPos++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(c.id)" + baseQuery + whereClause
	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count coupons failed: %w", err)
	}

	limit := params.Pagination.Limit
	if limit <= 0 {
		limit = 10
	}
	page := params.Pagination.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	limitPos := argPos
	offsetPos := argPos + 1
	args = append(args, limit, offset)

	selectQuery := `
		SELECT
			c.id, c.code, c.description, c.type, c.discount_value, c.min_spend, c.max_discount,
			c.quota_total, c.quota_remaining, c.starts_at, c.expires_at, c.is_active,
			c.created_at, c.updated_at
	` + baseQuery + whereClause +
		fmt.Sprintf(" ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d", limitPos, offsetPos)

	rows, err := exec.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list coupons failed: %w", err)
	}
	defer rows.Close()

	var coupons []discountdomain.Coupon
	for rows.Next() {
		var c discountdomain.Coupon
		if err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Description,
			&c.Type,
			&c.DiscountValue,
			&c.MinSpend,
			&c.MaxDiscount,
			&c.QuotaTotal,
			&c.QuotaRemaining,
			&c.StartsAt,
			&c.ExpiresAt,
			&c.IsActive,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan coupon failed: %w", err)
		}
		coupons = append(coupons, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate coupons rows failed: %w", err)
	}

	return coupons, total, nil
}

func (r *CouponRepository) DecrementQuota(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE coupons
		SET quota_remaining = quota_remaining - 1,
		    updated_at = NOW()
		WHERE id = $1 AND quota_remaining > 0 AND is_active = true
	`

	res, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("decrement coupon quota failed: %w", err)
	}

	if res.RowsAffected() == 0 {
		return discountdomain.ErrCouponQuotaExceeded
	}

	return nil
}

func (r *CouponRepository) SaveRedemption(
	ctx context.Context,
	exec transaction.Executor,
	redemption discountdomain.CouponRedemption,
) error {
	query := `
		INSERT INTO coupon_redemptions (
			id, coupon_id, customer_id, order_id, discount_amount, redeemed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, err := exec.Exec(
		ctx,
		query,
		redemption.ID,
		redemption.CouponID,
		redemption.CustomerID,
		redemption.OrderID,
		redemption.DiscountAmount,
		redemption.RedeemedAt,
	)
	if err != nil {
		return fmt.Errorf("save coupon redemption failed: %w", err)
	}

	return nil
}

func (r *CouponRepository) GetRedemptionByOrder(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) (*discountdomain.CouponRedemption, error) {
	query := `
		SELECT id, coupon_id, customer_id, order_id, discount_amount, redeemed_at
		FROM coupon_redemptions
		WHERE order_id = $1
		LIMIT 1
	`

	var red discountdomain.CouponRedemption
	err := exec.QueryRow(ctx, query, orderID).Scan(
		&red.ID,
		&red.CouponID,
		&red.CustomerID,
		&red.OrderID,
		&red.DiscountAmount,
		&red.RedeemedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, transaction.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get redemption by order failed: %w", err)
	}

	return &red, nil
}
