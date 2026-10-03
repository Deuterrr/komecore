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

type shopAddressRepositoryImpl struct{}

func NewShopAddressRepositoryImpl() repository.ShopAddressRepository {
	return &shopAddressRepositoryImpl{}
}

func (r *shopAddressRepositoryImpl) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) (*domain.ShopAddress, error) {
	query := `
		SELECT
			id,
			shop_id,
			label,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_active,
			created_at,
			updated_at
		FROM shop_addresses
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	var (
		a             domain.ShopAddress
		unusedVillage string
	)
	err := exec.QueryRow(ctx, query, addressID).Scan(
		&a.ID,
		&a.ShopID,
		&a.Label,
		&a.Phone,
		&a.Detail.Province,
		&a.Detail.City,
		&a.Detail.District,
		&unusedVillage,
		&a.Detail.FullAddress,
		&a.Detail.PostalCode,
		&a.Detail.Latitude,
		&a.Detail.Longitude,
		&a.IsActive,
		&a.CreatedAt,
		&a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query address by id failed: %w", err)
	}

	return &a, nil
}

func (r *shopAddressRepositoryImpl) GetDefaultByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) (*domain.ShopAddress, error) {
	query := `
		SELECT
			id,
			shop_id,
			label,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_active,
			created_at,
			updated_at
		FROM
			shop_addresses
		WHERE
			shop_id = $1
			AND is_active = true
			AND deleted_at IS NULL
		LIMIT 1
	`

	var (
		a             domain.ShopAddress
		unusedVillage string
	)
	err := exec.QueryRow(ctx, query, shopID).Scan(
		&a.ID,
		&a.ShopID,
		&a.Label,
		&a.Phone,
		&a.Detail.Province,
		&a.Detail.City,
		&a.Detail.District,
		&unusedVillage,
		&a.Detail.FullAddress,
		&a.Detail.PostalCode,
		&a.Detail.Latitude,
		&a.Detail.Longitude,
		&a.IsActive,
		&a.CreatedAt,
		&a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query address default by shop id failed: %w", err)
	}

	return &a, nil
}

func (r *shopAddressRepositoryImpl) GetDefaultsByShopIDs(
	ctx context.Context,
	exec transaction.Executor,
	shopIDs []uuid.UUID,
) (map[uuid.UUID]domain.ShopAddress, error) {
	addressMap := make(map[uuid.UUID]domain.ShopAddress)
	if len(shopIDs) == 0 {
		return addressMap, nil
	}

	shopIDStrings := make([]string, len(shopIDs))
	for i, id := range shopIDs {
		shopIDStrings[i] = id.String()
	}

	query := `
		SELECT
			id,
			shop_id,
			label,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_active,
			created_at,
			updated_at
		FROM
			shop_addresses
		WHERE
			shop_id = ANY($1::uuid[])
			AND is_active = true
			AND deleted_at IS NULL
	`

	rows, err := exec.Query(ctx, query, shopIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query shop addresses by shop ids failed: %w", err)
	}
	defer rows.Close()

	var shopAddresses []domain.ShopAddress
	for rows.Next() {
		var (
			sC            domain.ShopAddress
			unusedVillage string
		)
		if err := rows.Scan(
			&sC.ID,
			&sC.ShopID,
			&sC.Label,
			&sC.Phone,
			&sC.Detail.Province,
			&sC.Detail.City,
			&sC.Detail.District,
			&unusedVillage,
			&sC.Detail.FullAddress,
			&sC.Detail.PostalCode,
			&sC.Detail.Latitude,
			&sC.Detail.Longitude,
			&sC.IsActive,
			&sC.CreatedAt,
			&sC.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("mapping shop address model to domain failed: %w", err)
		}
		shopAddresses = append(shopAddresses, sC)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shop addresses failed: %w", err)
	}

	for _, shopAddr := range shopAddresses {
		addressMap[shopAddr.ShopID] = shopAddr
	}

	return addressMap, nil
}

func (r *shopAddressRepositoryImpl) FindByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) ([]domain.ShopAddress, error) {
	query := `
		SELECT
			id,
			shop_id,
			label,
			phone,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			is_active,
			created_at,
			updated_at
		FROM shop_addresses
		WHERE shop_id = $1 AND deleted_at IS NULL
	`

	rows, err := exec.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("query address by shop id failed: %w", err)
	}
	defer rows.Close()

	var addresses []domain.ShopAddress
	for rows.Next() {
		var (
			a             domain.ShopAddress
			unusedVillage string
		)

		err := rows.Scan(
			&a.ID,
			&a.ShopID,
			&a.Label,
			&a.Phone,
			&a.Detail.Province,
			&a.Detail.City,
			&a.Detail.District,
			&unusedVillage,
			&a.Detail.FullAddress,
			&a.Detail.PostalCode,
			&a.Detail.Latitude,
			&a.Detail.Longitude,
			&a.IsActive,
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

func (r *shopAddressRepositoryImpl) Create(
	ctx context.Context,
	exec transaction.Executor,
	address domain.ShopAddress,
) error {
	query := `
		INSERT INTO shop_addresses (
			id,
			shop_id,
			label,
			phone,
			is_active,
			province,
			city,
			district,
			village,
			full_address,
			postal_code,
			latitude,
			longitude,
			created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		)
	`

	_, err := exec.Exec(ctx, query,
		address.ID,
		address.ShopID,
		address.Label,
		address.Phone,
		address.IsActive,
		address.Detail.Province,
		address.Detail.City,
		address.Detail.District,
		"",
		address.Detail.FullAddress,
		address.Detail.PostalCode,
		address.Detail.Latitude,
		address.Detail.Longitude,
		address.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert address failed: %w", err)
	}

	return nil
}

func (r *shopAddressRepositoryImpl) Update(
	ctx context.Context,
	exec transaction.Executor,
	address domain.ShopAddress,
) error {
	query := `
		UPDATE shop_addresses
		SET
			label = $1,
			phone = $2,
			is_active = $3,
			province = $4,
			city = $5,
			district = $6,
			village = $7,
			full_address = $8,
			postal_code = $9,
			latitude = $10,
			longitude = $11,
			updated_at = NOW()
		WHERE
			id = $12
			AND shop_id = $13
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query,
		address.Label,
		address.Phone,
		address.IsActive,
		address.Detail.Province,
		address.Detail.City,
		address.Detail.District,
		"",
		address.Detail.FullAddress,
		address.Detail.PostalCode,
		address.Detail.Latitude,
		address.Detail.Longitude,
		address.ID,
		address.ShopID,
	)
	if err != nil {
		return fmt.Errorf("update address failed: %w", err)
	}

	return nil
}

func (r *shopAddressRepositoryImpl) Delete(
	ctx context.Context,
	exec transaction.Executor,
	addressID uuid.UUID,
) error {
	query := `
		UPDATE shop_addresses
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, addressID)
	if err != nil {
		return fmt.Errorf("delete address failed: %w", err)
	}

	return nil
}

func (r *shopAddressRepositoryImpl) UnsetActiveByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) error {
	query := `
		UPDATE shop_addresses
		SET
			is_active = false,
			updated_at = NOW()
		WHERE
			shop_id = $1
			AND is_active = true
			AND deleted_at IS NULL
	`

	_, err := exec.Exec(ctx, query, shopID)
	if err != nil {
		return fmt.Errorf("unset active shop addresses failed: %w", err)
	}

	return nil
}
