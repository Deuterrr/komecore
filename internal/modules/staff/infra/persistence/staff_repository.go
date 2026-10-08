package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/domain"
	"komecore/internal/modules/staff/repository"
	query "komecore/internal/shared/query"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type StaffRepository struct{}

func NewStaffRepository() *StaffRepository {
	return &StaffRepository{}
}

func (r *StaffRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	staff domain.Staff,
) error {
	query := `
		INSERT INTO staff (
			id,
			user_id,
			created_at
		) VALUES ($1,$2,$3)
	`

	_, err := exec.Exec(ctx, query,
		staff.ID,
		staff.UserID,
		staff.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert staff failed: %w", err)
	}
	return nil
}

func (r *StaffRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*domain.Staff, error) {
	query := `
		SELECT
			id,
			user_id,
			created_at,
			updated_at
		FROM staff
		WHERE id = $1
			AND deleted_at IS NULL
		LIMIT 1
	`

	var m domain.Staff
	err := exec.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.UserID,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query staff by id failed: %w", err)
	}
	return &m, nil
}

func (r *StaffRepository) GetProfileByUserID(
	ctx context.Context,
	exec transaction.Executor,
	userID uuid.UUID,
) (*domain.StaffProfile, error) {
	query := `
		SELECT
			s.id,
			s.user_id,
			u.name,
			u.username,
			u.phone,
			u.avatar_url,
			s.created_at,
			s.updated_at
		FROM staff s
		INNER JOIN users u
			ON u.id = s.user_id
		WHERE s.user_id = $1
	`

	var profile domain.StaffProfile
	err := exec.QueryRow(ctx, query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Name,
		&profile.Username,
		&profile.Phone,
		&profile.AvatarURL,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFoundStaff
		}

		return nil, fmt.Errorf("get staff profile by user id failed: %w", err)
	}

	return &profile, nil
}

func (r *StaffRepository) FindStaff(
	ctx context.Context,
	exec transaction.Executor,
	params repository.FindStaffParams,
) ([]domain.StaffProfile, int, error) {
	baseQuery := `
		FROM staff m
		INNER JOIN users u ON u.id = m.user_id
	`

	selectQuery := `
		SELECT
			m.id,
			m.user_id,
			u.name,
			u.username,
			u.phone,
			u.avatar_url,
			m.created_at,
			m.updated_at
	`

	// Build filters
	// Apply search criteria and soft-delete constraints
	whereClause := ""
	notDeletedCondition := "m.deleted_at IS NULL AND u.deleted_at IS NULL"

	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	conditions = append(conditions, notDeletedCondition)

	if params.ID != nil {
		conditions = append(conditions, fmt.Sprintf("m.id = $%d", argPos))
		args = append(args, *params.ID)
		argPos++
	}

	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count matching products
	// Used for pagination metadata
	countQuery := `
		SELECT COUNT(*)
	` + baseQuery + whereClause

	countArgs := append([]any{}, args...)

	var total int
	err := exec.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("query count staff failed: %w", err)
	}

	// Build sorting expressions
	// Convert requested sort keys into SQL ORDER BY clauses
	var staffSortKeys = map[query.SortKey]string{
		repository.StaffSortLatest: "m.created_at",
		repository.StaffSortModify: "m.updated_at",
	}

	var sortClauses []string
	for _, sort := range params.Sorts {
		colName, exists := staffSortKeys[sort.By]
		if !exists {
			continue
		}

		dir := "DESC"
		if sort.Direction == query.SortAsc {
			dir = "ASC"
		}

		sortClauses = append(
			sortClauses,
			fmt.Sprintf("%s %s", colName, dir),
		)
	}

	orderBy := "ORDER BY m.created_at DESC"
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
	queryStr := selectQuery + baseQuery + whereClause + " " + orderBy +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPos, offsetPos)

	rows, err := exec.Query(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query list staff failed: %w", err)
	}
	defer rows.Close()

	var results []domain.StaffProfile
	for rows.Next() {
		var m domain.StaffProfile
		err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.Name,
			&m.Username,
			&m.Phone,
			&m.AvatarURL,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("mapping staff model to domain failed: %w", err)
		}
		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate staff failed: %w", err)
	}

	return results, total, nil
}

func (r *StaffRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	staffID uuid.UUID,
	name string,
	logoUrl *string,
	bannerUrl *string,
) error {
	staffQuery := `
		UPDATE staff
		SET updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	staffRes, err := exec.Exec(ctx, staffQuery, staffID)
	if err != nil {
		return fmt.Errorf("update staff failed: %w", err)
	}
	if staffRes.RowsAffected() == 0 {
		return domain.ErrNotFoundStaff
	}

	userQuery := `
		UPDATE users
		SET name = $2,
		    avatar_url = COALESCE($3, avatar_url),
		    updated_at = NOW()
		WHERE id = (SELECT user_id FROM staff WHERE id = $1)
		  AND deleted_at IS NULL
	`
	_, err = exec.Exec(ctx, userQuery,
		staffID,
		name,
		logoUrl,
	)
	if err != nil {
		return fmt.Errorf("update staff user failed: %w", err)
	}

	return nil
}

func (r *StaffRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	staffID uuid.UUID,
) error {
	query := `
		UPDATE staff
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`
	res, err := exec.Exec(ctx, query, staffID)
	if err != nil {
		return fmt.Errorf("soft delete staff failed: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrNotFoundStaff
	}

	return nil
}
