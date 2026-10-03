package persistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/address/domain"
	"komecore/internal/modules/address/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type customerAddressRepositoryImpl struct{}

func NewCustomerAddressRepositoryImpl() repository.CustomerAddressRepository {
	return &customerAddressRepositoryImpl{}
}

func (r *customerAddressRepositoryImpl) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) (*domain.CustomerAddress, error) {
	query := `
		SELECT
			id,
			customer_id,
			recipient_name,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_default,
			created_at,
			updated_at
		FROM customer_addresses
		WHERE id = $1 AND deleted_at IS NULL
	`

	var (
		add           domain.CustomerAddress
		unusedVillage string
	)
	err := exec.QueryRow(ctx, query, addressID).Scan(
		&add.ID,
		&add.CustomerID,
		&add.ReceiverName,
		&add.Phone,
		&add.Detail.Province,
		&add.Detail.City,
		&add.Detail.District,
		&unusedVillage,
		&add.Detail.FullAddress,
		&add.Detail.PostalCode,
		&add.Detail.Latitude,
		&add.Detail.Longitude,
		&add.IsDefault,
		&add.CreatedAt,
		&add.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query get address by id failed: %w", err)
	}

	return &add, nil
}

func (r *customerAddressRepositoryImpl) GetDefaultByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*domain.CustomerAddress, error) {
	query := `
		SELECT
			id,
			customer_id,
			recipient_name,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_default,
			created_at,
			updated_at
		FROM
			customer_addresses
		WHERE
			customer_id = $1
			AND is_default = true
			AND deleted_at IS NULL
		LIMIT 1
	`

	var (
		add           domain.CustomerAddress
		unusedVillage string
	)
	err := exec.QueryRow(ctx, query, customerID).Scan(
		&add.ID,
		&add.CustomerID,
		&add.ReceiverName,
		&add.Phone,
		&add.Detail.Province,
		&add.Detail.City,
		&add.Detail.District,
		&unusedVillage,
		&add.Detail.FullAddress,
		&add.Detail.PostalCode,
		&add.Detail.Latitude,
		&add.Detail.Longitude,
		&add.IsDefault,
		&add.CreatedAt,
		&add.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query get default address by customer id failed: %w", err)
	}

	return &add, nil
}

func (r *customerAddressRepositoryImpl) ListByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) ([]domain.CustomerAddress, error) {
	query := `
		SELECT
			id,
			customer_id,
			recipient_name,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_default,
			created_at,
			updated_at
		FROM customer_addresses
		WHERE customer_id = $1 AND deleted_at IS NULL
	`

	rows, err := exec.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("query customer_addresses by customer id failed: %w", err)
	}
	defer rows.Close()

	var addresses []domain.CustomerAddress
	for rows.Next() {
		var (
			a             domain.CustomerAddress
			unusedVillage string
		)

		err := rows.Scan(
			&a.ID,
			&a.CustomerID,
			&a.ReceiverName,
			&a.Phone,
			&a.Detail.Province,
			&a.Detail.City,
			&a.Detail.District,
			&unusedVillage,
			&a.Detail.FullAddress,
			&a.Detail.PostalCode,
			&a.Detail.Latitude,
			&a.Detail.Longitude,
			&a.IsDefault,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping address model to domain failed: %w", err)
		}
		addresses = append(addresses, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate addresses failed: %w", err)
	}

	return addresses, nil
}

func (r *customerAddressRepositoryImpl) CountByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) (*int, error) {
	query := `
		SELECT COUNT(*)
		FROM
			customer_addresses
		WHERE
			customer_id = $1
			AND deleted_at IS NULL
	`

	var count int
	err := exec.QueryRow(ctx, query, customerID).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query count address failed: %w", err)
	}

	return &count, nil
}

func (r *customerAddressRepositoryImpl) UnsetDefaultByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) error {
	query := `
		UPDATE
			customer_addresses
		SET
			is_default = false,
			updated_at = NOW()
		WHERE
			customer_id = $1
			AND is_default = true
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("query unset default address failed: %w", err)
	}

	return nil
}

func (r *customerAddressRepositoryImpl) Save(
	ctx context.Context,
	exec transaction.Executor,
	address domain.CustomerAddress,
) error {
	query := `
		INSERT INTO customer_addresses (
			id,
			customer_id,
			recipient_name,
			phone,
			is_default,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			created_at,
			updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15
		)
		ON CONFLICT (id)
		DO UPDATE SET
			recipient_name = EXCLUDED.recipient_name,
			phone = EXCLUDED.phone,
			is_default = EXCLUDED.is_default,
			province = EXCLUDED.province,
			city = EXCLUDED.city,
			district = EXCLUDED.district,
			village = EXCLUDED.village,
			full_address = EXCLUDED.full_address,
			postal_code = EXCLUDED.postal_code,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			updated_at = EXCLUDED.updated_at
	`

	_, err := exec.Exec(ctx, query,
		address.ID,
		address.CustomerID,
		address.ReceiverName,
		address.Phone,
		address.IsDefault,
		address.Detail.Province,
		address.Detail.City,
		address.Detail.District,
		"",
		address.Detail.FullAddress,
		address.Detail.PostalCode,
		address.Detail.Latitude,
		address.Detail.Longitude,
		address.CreatedAt,
		address.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert address failed: %w", err)
	}

	return nil
}

func (r *customerAddressRepositoryImpl) Delete(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) error {
	query := `
		UPDATE
			customer_addresses
		SET
			is_default = false,
			deleted_at = now()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query,
		addressID,
	)
	if err != nil {
		return fmt.Errorf("delete address failed: %w", err)
	}

	return nil
}

func (r *customerAddressRepositoryImpl) DeleteByCustomerID(
	ctx context.Context,
	exec transaction.Executor,
	customerID uuid.UUID,
) error {
	query := `
		UPDATE
			customer_addresses
		SET
			is_default = false,
			deleted_at = now()
		WHERE
			customer_id = $1
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("delete customer addresses failed: %w", err)
	}

	return nil
}

func (r *customerAddressRepositoryImpl) ListByIDs(
	ctx context.Context,
	exec transaction.Executor,
	addressIDs []uuid.UUID,
) ([]domain.CustomerAddress, error) {
	if len(addressIDs) == 0 {
		return []domain.CustomerAddress{}, nil
	}

	query := `
		SELECT
			id,
			customer_id,
			recipient_name,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_default,
			created_at,
			updated_at
		FROM customer_addresses
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL
	`

	addressIDStrings := make([]string, len(addressIDs))
	for i, id := range addressIDs {
		addressIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, addressIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query customer addresses by ids failed: %w", err)
	}
	defer rows.Close()

	var addresses []domain.CustomerAddress
	for rows.Next() {
		var (
			a             domain.CustomerAddress
			unusedVillage string
		)
		err := rows.Scan(
			&a.ID,
			&a.CustomerID,
			&a.ReceiverName,
			&a.Phone,
			&a.Detail.Province,
			&a.Detail.City,
			&a.Detail.District,
			&unusedVillage,
			&a.Detail.FullAddress,
			&a.Detail.PostalCode,
			&a.Detail.Latitude,
			&a.Detail.Longitude,
			&a.IsDefault,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("mapping address model to domain failed: %w", err)
		}
		addresses = append(addresses, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate addresses failed: %w", err)
	}

	return addresses, nil
}
