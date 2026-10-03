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

type staffMembershipRepositoryImpl struct{}

func NewStaffMembershipRepositoryImpl() repository.StaffMembershipRepository {
	return &staffMembershipRepositoryImpl{}
}

func (r *staffMembershipRepositoryImpl) GetByAccountID(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
) (*domain.StaffMembership, error) {
	query := `
		SELECT
			id,
			staff_id,
			account_id,
			role_id,
			created_by,
			created_at
		FROM
			staff_memberships
		WHERE
			account_id = $1
		LIMIT 1
	`

	var m domain.StaffMembership
	err := exec.QueryRow(ctx, query, accountID).Scan(
		&m.ID,
		&m.StaffID,
		&m.AccountID,
		&m.RoleID,
		&m.CreatedBy,
		&m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query staff membership failed: %w", err)
	}

	return &m, nil
}

func (r *staffMembershipRepositoryImpl) GetByAccountIDAndStaffID(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
	staffID uuid.UUID,
) (*domain.StaffMembership, error) {
	query := `
		SELECT
			id,
			staff_id,
			account_id,
			role_id,
			created_by,
			created_at
		FROM
			staff_memberships
		WHERE
			account_id = $1 AND staff_id = $2
		LIMIT 1
	`

	var m domain.StaffMembership
	err := exec.QueryRow(ctx, query, accountID, staffID).Scan(
		&m.ID,
		&m.StaffID,
		&m.AccountID,
		&m.RoleID,
		&m.CreatedBy,
		&m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query staff membership by account and staff failed: %w", err)
	}

	return &m, nil
}

func (r *staffMembershipRepositoryImpl) ListRolesByAccountIDAndStaffID(
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
		JOIN
			roles ro ON ro.id = sm.role_id
		WHERE
			sm.account_id = $1 AND sm.staff_id = $2
	`

	rows, err := exec.Query(ctx, query, accountID, staffID)
	if err != nil {
		return nil, fmt.Errorf("query roles by membership failed: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(
			&role.ID,
			&role.Code,
			&role.Name,
		); err != nil {
			return nil, fmt.Errorf("scan role failed: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles failed: %w", err)
	}

	return roles, nil
}

func (r *staffMembershipRepositoryImpl) Save(
	ctx context.Context,
	exec transaction.Executor,
	membership domain.StaffMembership,
) error {
	query := `
		INSERT INTO staff_memberships (
			id,
			staff_id,
			account_id,
			role_id,
			created_by,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account_id)
		DO UPDATE SET
			role_id = EXCLUDED.role_id,
			staff_id = EXCLUDED.staff_id
	`

	_, err := exec.Exec(
		ctx,
		query,
		membership.ID,
		membership.StaffID,
		membership.AccountID,
		membership.RoleID,
		membership.CreatedBy,
		membership.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save staff membership failed: %w", err)
	}

	return nil
}

func (r *staffMembershipRepositoryImpl) DeleteByAccountID(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
) error {
	query := `
		DELETE FROM staff_memberships
		WHERE account_id = $1
	`

	_, err := exec.Exec(ctx, query, accountID)
	if err != nil {
		return fmt.Errorf("delete staff membership by account id failed: %w", err)
	}

	return nil
}

func (r *staffMembershipRepositoryImpl) DeleteByAccountIDAndStaffID(
	ctx context.Context,
	exec transaction.Executor,
	accountID uuid.UUID,
	staffID uuid.UUID,
) error {
	query := `
		DELETE FROM staff_memberships
		WHERE account_id = $1 AND staff_id = $2
	`

	_, err := exec.Exec(ctx, query, accountID, staffID)
	if err != nil {
		return fmt.Errorf("delete staff membership by account and staff id failed: %w", err)
	}

	return nil
}

func (r *staffMembershipRepositoryImpl) DeleteByStaffID(
	ctx context.Context,
	exec transaction.Executor,
	staffID uuid.UUID,
) error {
	query := `
		DELETE FROM staff_memberships
		WHERE staff_id = $1
	`

	_, err := exec.Exec(ctx, query, staffID)
	if err != nil {
		return fmt.Errorf("delete staff membership by staff id failed: %w", err)
	}

	return nil
}

func (r *staffMembershipRepositoryImpl) ListAccountsByStaffID(
	ctx context.Context,
	exec transaction.Executor,
	staffID uuid.UUID,
) ([]domain.StaffAccountMember, error) {
	query := `
		SELECT
			sm.id,
			sm.account_id,
			acc.email,
			ro.id,
			ro.code,
			ro.name,
			sm.created_by,
			sm.created_at
		FROM
			staff_memberships sm
		JOIN
			accounts acc ON acc.id = sm.account_id
		JOIN
			roles ro ON ro.id = sm.role_id
		WHERE
			sm.staff_id = $1
		ORDER BY
			sm.created_at ASC
	`

	rows, err := exec.Query(ctx, query, staffID)
	if err != nil {
		return nil, fmt.Errorf("query accounts by staff id failed: %w", err)
	}
	defer rows.Close()

	var accounts []domain.StaffAccountMember
	for rows.Next() {
		var (
			a    domain.StaffAccountMember
			role domain.Role
		)
		if err := rows.Scan(
			&a.ID,
			&a.AccountID,
			&a.Email,
			&role.ID,
			&role.Code,
			&role.Name,
			&a.CreatedBy,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan staff account member failed: %w", err)
		}
		a.Role = role
		accounts = append(accounts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staff account members failed: %w", err)
	}

	return accounts, nil
}
