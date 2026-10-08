package staffpersistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/staff/staffdomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RoleRepository struct{}

func NewRoleRepository() *RoleRepository {
	return &RoleRepository{}
}

func (r *RoleRepository) GetRolesByAccountAndStaff(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
	staffID uuid.UUID,
) ([]staffdomain.Role, error) {
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

	var roles []staffdomain.Role
	for rows.Next() {
		var role staffdomain.Role
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

func (r *RoleRepository) GetByCode(
	ctx context.Context,
	exec transaction.Executor,
	code staffdomain.RoleCode,
) (*staffdomain.Role, error) {
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

	var role staffdomain.Role
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
