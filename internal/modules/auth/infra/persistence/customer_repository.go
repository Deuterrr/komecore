package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/auth/domain"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CustomerRepository struct{}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{}
}

func (r *CustomerRepository) Create(ctx context.Context, exec transaction.Executor, customer domain.Customer) error {
	query := `
		INSERT INTO customers (
			id,
			user_id,
			created_at
		) VALUES ($1,$2,$3)
	`

	_, err := exec.Exec(ctx, query,
		customer.ID,
		customer.UserID,
		customer.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert customer failed: %w", err)
	}
	return nil
}

func (r *CustomerRepository) GetByUserID(ctx context.Context, exec transaction.Executor, userID uuid.UUID) (*domain.Customer, error) {
	query := `
		SELECT
			id,
			user_id,
			created_at,
			updated_at
		FROM
			customers
		WHERE
			user_id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var c domain.Customer
	err := exec.QueryRow(ctx, query, userID).Scan(
		&c.ID,
		&c.UserID,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get customer by user id: %w", err)
	}

	return &c, nil
}

func (r *CustomerRepository) Delete(ctx context.Context, exec transaction.Executor, id uuid.UUID) error {
	query := `
		UPDATE customers
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, appclock.Now(), id)
	if err != nil {
		return fmt.Errorf("soft delete customer failed: %w", err)
	}

	return nil
}
