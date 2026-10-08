package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/product/domain"
	"komecore/internal/modules/product/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) FindProducts(
	ctx context.Context,
	exec transaction.Executor,
	params repository.FindProductParams,
) ([]domain.Product, int, error) {
	baseQuery := `
		FROM products p
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
			p.average_rating,
			p.review_count,
			p.created_at,
			p.updated_at,
			p.archived_at
	`

	// Build filters
	// Apply search criteria and soft-delete constraints
	whereClause := ""
	notDeletedCondition := "p.deleted_at IS NULL"

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	conditions = append(conditions, notDeletedCondition)

	if params.ShopID != nil {
		baseQuery += " INNER JOIN inventory i ON p.id = i.product_id AND i.deleted_at IS NULL"
		conditions = append(
			conditions,
			fmt.Sprintf("i.shop_id = $%d", argPos),
		)
		args = append(args, *params.ShopID)
		argPos++
	} else if params.ShopSlug != nil {
		baseQuery += " INNER JOIN inventory i ON p.id = i.product_id AND i.deleted_at IS NULL"
		conditions = append(
			conditions,
			fmt.Sprintf("i.shop_id IN (SELECT id FROM shops WHERE slug = $%d AND deleted_at IS NULL)", argPos),
		)
		args = append(args, *params.ShopSlug)
		argPos++
	}

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

	if params.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.status = $%d", argPos),
		)
		args = append(args, *params.Status)
		argPos++
	} else if params.ExcludeArchived {
		conditions = append(conditions, "p.status != 'archived'")
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count matching products
	// Used for pagination metadata
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
	` + baseQuery + whereClause

	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count products failed: %w", err)
	}

	// Build sorting expressions
	// Convert requested sort keys into SQL ORDER BY clauses
	var productSortKeys = map[query.SortKey]string{
		repository.ProductSortLatest:   "created_at",
		repository.ProductSortName:     "name",
		repository.ProductSortPrice:    "base_price",
		repository.ProductSortWeight:   "weight",
		repository.ProductSortStatus:   "status",
		repository.ProductSortModified: "updated_at",
		repository.ProductSortArchived: "archived_at",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := productSortKeys[sort.By]
		if !exists {
			continue
		}

		direction := "DESC"
		if sort.Direction == query.SortAsc {
			direction = "ASC"
		}

		sortClauses = append(
			sortClauses,
			fmt.Sprintf("p.%s %s", colName, direction),
		)
	}

	orderBy := "ORDER BY p.created_at DESC"
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
	query := selectQuery +
		baseQuery +
		whereClause +
		" " +
		orderBy +
		fmt.Sprintf(
			" LIMIT $%d OFFSET $%d",
			limitPos,
			offsetPos,
		)

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query products failed: %w", err)
	}
	defer rows.Close()

	var results []domain.Product
	for rows.Next() {
		var item domain.Product
		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.Status,
			&item.Price,
			&item.Weight,
			&item.AverageRating,
			&item.ReviewCount,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ArchivedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"mapping product model to domain failed: %w",
				err,
			)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products failed: %w", err)
	}

	return results, total, nil
}

func (r *ProductRepository) FindProductsWithInventory(
	ctx context.Context,
	exec transaction.Executor,
	params repository.FindProductParams,
) ([]domain.ProductWithInventory, int, error) {
	baseQuery := `
		FROM products p
		LEFT JOIN inventory i ON p.id = i.product_id
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
			p.average_rating,
			p.review_count,
			p.created_at,
			p.updated_at,
			p.archived_at,
			COALESCE(SUM(i.stock), 0)::integer AS total_stock,
			COALESCE(SUM(i.reserved_stock), 0)::integer AS total_reserved_stock
	`

	// Build filters
	// Apply search criteria and soft-delete constraints
	whereClause := ""
	notDeletedCondition := "p.deleted_at IS NULL"

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	conditions = append(conditions, notDeletedCondition)

	if params.ShopID != nil {
		shopIDCondition := fmt.Sprintf("i.shop_id = $%d", argPos)
		conditions = append(
			conditions,
			shopIDCondition,
		)
		args = append(args, *params.ShopID)
		argPos++
	} else if params.ShopSlug != nil {
		shopSlugCondition := fmt.Sprintf("i.shop_id IN (SELECT id FROM shops WHERE slug = $%d AND deleted_at IS NULL)", argPos)
		conditions = append(
			conditions,
			shopSlugCondition,
		)
		args = append(args, *params.ShopSlug)
		argPos++
	}

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

	if params.Status != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("p.status = $%d", argPos),
		)
		args = append(args, *params.Status)
		argPos++
	} else if params.ExcludeArchived {
		conditions = append(conditions, "p.status != 'archived'")
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count matching products
	// Used for pagination metadata
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
	` + baseQuery + whereClause

	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count products failed: %w", err)
	}

	// Build sorting expressions
	// Convert requested sort keys into SQL ORDER BY clauses
	var productSortKeys = map[query.SortKey]string{
		repository.ProductSortLatest:   "p.created_at",
		repository.ProductSortName:     "p.name",
		repository.ProductSortPrice:    "p.base_price",
		repository.ProductSortWeight:   "p.weight",
		repository.ProductSortStatus:   "p.status",
		repository.ProductSortModified: "p.updated_at",
		repository.ProductSortArchived: "p.archived_at",
		repository.ProductSortStock:    "COALESCE(SUM(i.stock), 0)",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := productSortKeys[sort.By]
		if !exists {
			continue
		}

		direction := "DESC"
		if sort.Direction == query.SortAsc {
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
	query := selectQuery +
		baseQuery +
		whereClause +
		" GROUP BY p.id " +
		orderBy +
		fmt.Sprintf(
			" LIMIT $%d OFFSET $%d",
			limitPos,
			offsetPos,
		)

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query products failed: %w", err)
	}
	defer rows.Close()

	var results []domain.ProductWithInventory
	for rows.Next() {
		var item domain.ProductWithInventory
		err := rows.Scan(
			&item.Product.ID,
			&item.Product.SKU,
			&item.Product.Name,
			&item.Product.Slug,
			&item.Product.Description,
			&item.Product.Status,
			&item.Product.Price,
			&item.Product.Weight,
			&item.Product.AverageRating,
			&item.Product.ReviewCount,
			&item.Product.CreatedAt,
			&item.Product.UpdatedAt,
			&item.Product.ArchivedAt,
			&item.TotalStock,
			&item.ReservedStock,
		)
		if err != nil {
			return nil, 0, fmt.Errorf(
				"mapping product model to domain failed: %w",
				err,
			)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate products failed: %w", err)
	}

	return results, total, nil
}

func (r *ProductRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.Product, error) {
	query := `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.slug,
			p.description,
			p.status,
			p.base_price,
			p.weight,
			p.average_rating,
			p.review_count,
			p.created_at,
			p.updated_at,
			p.archived_at
		FROM products p
		WHERE p.id = $1 AND p.deleted_at IS NULL
		LIMIT 1
	`

	var result domain.Product
	err := exec.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.SKU,
		&result.Name,
		&result.Slug,
		&result.Description,
		&result.Status,
		&result.Price,
		&result.Weight,
		&result.AverageRating,
		&result.ReviewCount,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.ArchivedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query product by id failed: %w", err)
	}

	return &result, nil
}

func (r *ProductRepository) GetBySlug(
	ctx context.Context,
	exec transaction.Executor,
	slug string,
) (*domain.Product, error) {
	query := `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.slug,
			p.description,
			p.status,
			p.base_price,
			p.weight,
			p.average_rating,
			p.review_count,
			p.created_at,
			p.updated_at,
			p.archived_at
		FROM products p
		WHERE p.slug = $1 AND p.deleted_at IS NULL
		LIMIT 1
	`

	var result domain.Product
	err := exec.QueryRow(ctx, query, slug).Scan(
		&result.ID,
		&result.SKU,
		&result.Name,
		&result.Slug,
		&result.Description,
		&result.Status,
		&result.Price,
		&result.Weight,
		&result.AverageRating,
		&result.ReviewCount,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.ArchivedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query product by id failed: %w", err)
	}

	return &result, nil
}

func (r *ProductRepository) FindByIDs(
	ctx context.Context,
	exec transaction.Executor,
	ids []uuid.UUID,
) ([]domain.Product, error) {
	if len(ids) == 0 {
		return []domain.Product{}, nil
	}

	query := `
		SELECT
			p.id,
			p.sku,
			p.name,
			p.slug,
			p.description,
			p.status,
			p.base_price,
			p.weight,
			p.average_rating,
			p.review_count,
			p.created_at,
			p.updated_at,
			p.archived_at
		FROM products p
		WHERE p.id = ANY($1::uuid[]) AND p.deleted_at IS NULL
	`

	productIDStrings := make([]string, len(ids))
	for i, id := range ids {
		productIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, productIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query products by many ids failed: %w", err)
	}
	defer rows.Close()

	var results []domain.Product
	for rows.Next() {
		var item domain.Product

		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.Status,
			&item.Price,
			&item.Weight,
			&item.AverageRating,
			&item.ReviewCount,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ArchivedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("mapping product model to domain failed: %w", err)
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products failed: %w", err)
	}

	return results, nil
}

func (r *ProductRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	product *domain.Product,
) error {
	query := `
		INSERT INTO products (
			id,
			sku,
			name,
			slug,
			description,
			status,
			base_price,
			weight,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id)
		DO UPDATE SET
			sku = EXCLUDED.sku,
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			base_price = EXCLUDED.base_price,
			weight = EXCLUDED.weight,
			updated_at = NOW()
	`

	_, err := exec.Exec(ctx, query,
		product.ID,
		product.SKU,
		product.Name,
		product.Slug,
		product.Description,
		product.Status,
		product.Price,
		product.Weight,
		product.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("save product failed: %w", err)
	}

	return nil
}

func (r *ProductRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) error {
	query := `
		UPDATE products
		SET
			deleted_at = NOW()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product failed: %w", err)
	}

	return nil
}

func (r *ProductRepository) UpdateRating(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	averageRating float64,
	reviewCount int,
) error {
	query := `
		UPDATE products
		SET
			average_rating = $1,
			review_count = $2,
			updated_at = NOW()
		WHERE
			id = $3
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, averageRating, reviewCount, productID)
	if err != nil {
		return fmt.Errorf("update product rating failed: %w", err)
	}

	return nil
}
