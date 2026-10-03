package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/domain"
	"komecore/internal/modules/staff/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type roleRepositoryImpl struct{}

func NewRoleRepositoryImpl() repository.RoleRepository {
	return &roleRepositoryImpl{}
}

func (r *roleRepositoryImpl) GetRolesByAccountAndStaff(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
	staffID uuid.UUID,
) ([]domain.Role, error) {
	query := `
		SELECT
			ro.id,
			ro.code,
			ro.name
		FROM
			staff_memberships sm
		JOIN roles ro ON ro.id = sm.role_id
		WHERE
			sm.account_id = $1 AND sm.staff_id = $2
	`

	rows, err := exec.Query(ctx, query,
		accountID,
		staffID,
	)
	if err != nil {
		return nil, fmt.Errorf("query roles by membership failed: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name); err != nil {
			return nil, fmt.Errorf("scan role failed: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles failed: %w", err)
	}

	return roles, nil
}

func (r *roleRepositoryImpl) GetByCode(
	ctx context.Context,
	exec transaction.Executor,
	code domain.RoleCode,
) (*domain.Role, error) {
	query := `
		SELECT
			id,
			code,
			name
		FROM
			roles
		WHERE
			code = $1
		LIMIT 1
	`

	var role domain.Role
	err := exec.QueryRow(ctx, query, string(code)).Scan(
		&role.ID,
		&role.Code,
		&role.Name,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role by code failed: %w", err)
	}

	return &role, nil
}
