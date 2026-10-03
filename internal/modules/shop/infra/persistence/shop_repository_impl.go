package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shop/domain"
	"komecore/internal/modules/shop/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type shopRepositoryImpl struct{}

func NewShopRepositoryImpl() repository.ShopRepository {
	return &shopRepositoryImpl{}
}

func (r *shopRepositoryImpl) FindByParams(
	ctx context.Context,
	exec transaction.Executor,
	params repository.FindShopsParams,
) ([]domain.Shop, int, error) {
	baseQuery := `
		FROM shops s
	`

	selectQuery := `
		SELECT
			s.id,
			s.name,
			s.slug,
			s.description,
			s.is_active,
			s.approval_status,
			s.created_at,
			s.updated_at
	`

	// Build filters
	// Apply search criteria and soft-delete constraints
	whereClause := ""
	notDeletedCondition := "s.deleted_at IS NULL"

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	conditions = append(conditions, notDeletedCondition)

	if params.ID != nil {
		conditions = append(conditions, fmt.Sprintf("s.id = $%d", argPos))
		args = append(args, *params.ID)
		argPos++
	}

	if len(params.ShopIDs) > 0 {
		shopIDStrings := make([]string, len(params.ShopIDs))
		for i, id := range params.ShopIDs {
			shopIDStrings[i] = id.String()
		}
		conditions = append(conditions, fmt.Sprintf("s.id = ANY($%d::uuid[])", argPos))
		args = append(args, shopIDStrings)
		argPos++
	}

	if params.Name != nil {
		conditions = append(conditions, fmt.Sprintf("s.name ILIKE $%d", argPos))
		args = append(args, "%"+*params.Name+"%")
		argPos++
	}

	if params.Slug != nil {
		conditions = append(conditions, fmt.Sprintf("s.slug = $%d", argPos))
		args = append(args, *params.Slug)
		argPos++
	}

	if params.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("s.is_active = $%d", argPos))
		args = append(args, *params.IsActive)
		argPos++
	}

	if params.ApprovalStatus != nil {
		conditions = append(conditions, fmt.Sprintf("s.approval_status = $%d", argPos))
		args = append(args, *params.ApprovalStatus)
		argPos++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count matching products
	// Used for pagination metadata
	countQuery := `
		SELECT COUNT(DISTINCT s.id)
	` + baseQuery + whereClause

	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count shops failed: %w", err)
	}

	// Build sorting expressions
	// Convert requested sort keys into SQL ORDER BY clauses
	var shopSortKeys = map[query.SortKey]string{
		repository.ShopSortName:   "name",
		repository.ShopSortActive: "is_active",
		repository.ShopSortLatest: "created_at",
		repository.ShopSortModify: "updated_at",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := shopSortKeys[sort.By]
		if !exists {
			continue
		}

		dir := "DESC"
		if sort.Direction == query.SortAsc {
			dir = "ASC"
		}

		sortClauses = append(
			sortClauses,
			fmt.Sprintf("s.%s %s", colName, dir),
		)
	}

	orderBy := "ORDER BY s.created_at DESC"
	if len(sortClauses) > 0 {
		orderBy = "ORDER BY " + strings.Join(sortClauses, ", ")
	}

	// Apply pagination
	// Calculate limit and offset values
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
	query := selectQuery + baseQuery + whereClause + " " + orderBy +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPos, offsetPos)

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query list shops failed: %w", err)
	}
	defer rows.Close()

	var shops []domain.Shop
	for rows.Next() {
		var s domain.Shop
		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Slug,
			&s.Description,
			&s.IsActive,
			&s.ApprovalStatus,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("mapping shop model to domain failed: %w", err)
		}

		shops = append(shops, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate shops failed: %w", err)
	}

	return shops, total, nil
}

func (r *shopRepositoryImpl) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) (*domain.Shop, error) {
	query := `
		SELECT
			id,
			name,
			slug,
			description,
			is_active,
			approval_status,
			created_at,
			updated_at
		FROM shops
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var s domain.Shop
	err := exec.QueryRow(ctx, query, shopID).Scan(
		&s.ID,
		&s.Name,
		&s.Slug,
		&s.Description,
		&s.IsActive,
		&s.ApprovalStatus,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query shop by id failed: %w", err)
	}

	return &s, nil
}

func (r *shopRepositoryImpl) GetBySlug(
	ctx context.Context,
	exec transaction.Executor,
	slug string,
) (*domain.Shop, error) {
	query := `
		SELECT
			id,
			name,
			slug,
			description,
			is_active,
			approval_status,
			created_at,
			updated_at
		FROM shops
		WHERE slug = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var s domain.Shop
	err := exec.QueryRow(ctx, query, slug).Scan(
		&s.ID,
		&s.Name,
		&s.Slug,
		&s.Description,
		&s.IsActive,
		&s.ApprovalStatus,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query shop by slug failed: %w", err)
	}

	return &s, nil
}

func (r *shopRepositoryImpl) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	ids []uuid.UUID,
) ([]domain.Shop, error) {
	if len(ids) == 0 {
		return []domain.Shop{}, nil
	}

	query := `
		SELECT
			s.id,
			s.name,
			s.slug,
			s.description,
			s.is_active,
			s.approval_status,
			s.created_at,
			s.updated_at
		FROM shops s
		WHERE s.id = ANY($1::uuid[]) AND s.deleted_at IS NULL
	`

	shopIDStrings := make([]string, len(ids))
	for i, id := range ids {
		shopIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, shopIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query shops by many ids failed: %w", err)
	}
	defer rows.Close()

	var results []domain.Shop
	for rows.Next() {
		var s domain.Shop
		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Slug,
			&s.Description,
			&s.IsActive,
			&s.ApprovalStatus,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping shop model to domain failed: %w", err)
		}

		results = append(results, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shops failed: %w", err)
	}

	return results, nil
}

func (r *shopRepositoryImpl) Save(
	ctx context.Context,
	exec transaction.Executor,
	shop domain.Shop,
) error {
	approvalStatus := shop.ApprovalStatus
	if approvalStatus == "" {
		approvalStatus = domain.ShopApprovalStatusPending
	}

	query := `
		INSERT INTO shops (
			id,
			slug,
			name,
			description,
			is_active,
			approval_status,
			created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7
		)
		ON CONFLICT (id)
		DO UPDATE SET
			slug = EXCLUDED.slug,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			approval_status = EXCLUDED.approval_status,
			updated_at = NOW()
	`

	_, err := exec.Exec(ctx, query,
		shop.ID,
		shop.Slug,
		shop.Name,
		shop.Description,
		shop.IsActive,
		approvalStatus,
		shop.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert shop failed: %w", err)
	}
	return nil
}

func (r *shopRepositoryImpl) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE shops
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete shop failed: %w", err)
	}

	return nil
}
