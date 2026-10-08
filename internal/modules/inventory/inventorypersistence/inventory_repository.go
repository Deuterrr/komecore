package inventorypersistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InventoryRepository struct{}

func NewInventoryRepository() *InventoryRepository {
	return &InventoryRepository{}
}

func (r *InventoryRepository) GetByProductIDAndShopID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
) (*inventorydomain.Inventory, error) {
	query := `
		SELECT
			id,
			product_id,
			shop_id,
			stock,
			reserved_stock,
			created_at,
			updated_at
		FROM inventory
		WHERE product_id = $1 AND shop_id = $2
		LIMIT 1
	`

	var inventory inventorydomain.Inventory
	err := exec.QueryRow(ctx, query, productID, shopID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.ShopID,
		&inventory.TotalStock,
		&inventory.ReservedStock,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query inventory by product and shop failed: %w", err)
	}

	return &inventory, nil
}

func (r *InventoryRepository) ListByProductID(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
) ([]inventorydomain.Inventory, error) {
	query := `
		SELECT
			id,
			product_id,
			shop_id,
			stock,
			reserved_stock,
			created_at,
			updated_at
		FROM inventory
		WHERE product_id = $1
		ORDER BY created_at ASC
	`

	rows, err := exec.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("query inventory by product failed: %w", err)
	}
	defer rows.Close()

	var inventories []inventorydomain.Inventory
	for rows.Next() {
		var inventory inventorydomain.Inventory
		if err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.ShopID,
			&inventory.TotalStock,
			&inventory.ReservedStock,
			&inventory.CreatedAt,
			&inventory.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("mapping inventory model to domain failed: %w", err)
		}
		inventories = append(inventories, inventory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory failed: %w", err)
	}

	return inventories, nil
}

func (r *InventoryRepository) ListByProductIDs(
	ctx context.Context,
	exec transaction.Executor,
	productIDs []uuid.UUID,
) (map[uuid.UUID][]inventorydomain.Inventory, error) {
	result := make(map[uuid.UUID][]inventorydomain.Inventory)
	if len(productIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT
			id,
			product_id,
			shop_id,
			stock,
			reserved_stock,
			created_at,
			updated_at
		FROM inventory
		WHERE product_id = ANY($1::uuid[])
		ORDER BY created_at ASC
	`

	productIDStrings := make([]string, len(productIDs))
	for i, id := range productIDs {
		productIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, productIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query inventory by products failed: %w", err)
	}
	defer rows.Close()

	var inventories []inventorydomain.Inventory
	for rows.Next() {
		var inventory inventorydomain.Inventory
		if err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.ShopID,
			&inventory.TotalStock,
			&inventory.ReservedStock,
			&inventory.CreatedAt,
			&inventory.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("mapping inventory model to domain failed: %w", err)
		}
		inventories = append(inventories, inventory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory failed: %w", err)
	}

	for _, inventory := range inventories {
		result[inventory.ProductID] = append(result[inventory.ProductID], inventory)
	}

	return result, nil
}

func (r *InventoryRepository) ListByShopID(
	ctx context.Context,
	exec transaction.Executor,
	shopID uuid.UUID,
) ([]inventorydomain.Inventory, error) {
	query := `
		SELECT
			id,
			product_id,
			shop_id,
			stock,
			reserved_stock,
			created_at,
			updated_at
		FROM inventory
		WHERE shop_id = $1
		ORDER BY created_at ASC
	`

	rows, err := exec.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("query inventory by shop failed: %w", err)
	}
	defer rows.Close()

	var inventories []inventorydomain.Inventory
	for rows.Next() {
		var inventory inventorydomain.Inventory
		if err := rows.Scan(
			&inventory.ID,
			&inventory.ProductID,
			&inventory.ShopID,
			&inventory.TotalStock,
			&inventory.ReservedStock,
			&inventory.CreatedAt,
			&inventory.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("mapping inventory model to domain failed: %w", err)
		}
		inventories = append(inventories, inventory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory failed: %w", err)
	}

	return inventories, nil
}

func (r *InventoryRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	inventory *inventorydomain.Inventory,
) error {
	query := `
		INSERT INTO inventory (
			id,
			product_id,
			shop_id,
			stock,
			reserved_stock,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := exec.Exec(ctx, query,
		inventory.ID,
		inventory.ProductID,
		inventory.ShopID,
		inventory.TotalStock,
		inventory.ReservedStock,
		inventory.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert inventory failed: %w", err)
	}

	return nil
}

func (r *InventoryRepository) Reserve(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
	qty int,
) error {
	// Atomically increment reserved_stock only when there is enough
	// available stock (stock - reserved_stock >= qty)
	//
	// The UPDATE is its own implicit row-level lock, so no separate
	// SELECT FOR UPDATE is required
	query := `
		UPDATE inventory
		SET
			reserved_stock = reserved_stock + $1,
			updated_at     = NOW()
		WHERE
			product_id                    = $2
			AND shop_id                   = $3
			AND (stock - reserved_stock) >= $1
	`

	tag, err := exec.Exec(ctx, query,
		qty,
		productID,
		shopID,
	)
	if err != nil {
		return fmt.Errorf("reserve inventory failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return inventorydomain.ErrInsufficientStock
	}

	return nil
}

func (r *InventoryRepository) Release(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
	qty int,
) error {
	query := `
		UPDATE inventory
		SET
			reserved_stock = reserved_stock - $1,
			updated_at     = NOW()
		WHERE
			product_id       = $2
			AND shop_id      = $3
			AND reserved_stock >= $1
	`

	tag, err := exec.Exec(ctx, query,
		qty,
		productID,
		shopID,
	)
	if err != nil {
		return fmt.Errorf("release inventory failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return inventorydomain.ErrInsufficientReserved
	}

	return nil
}

func (r *InventoryRepository) Commit(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
	qty int,
) error {
	query := `
		UPDATE inventory
		SET
			stock          = stock - $1,
			reserved_stock = reserved_stock - $1,
			updated_at     = NOW()
		WHERE
			product_id       = $2
			AND shop_id      = $3
			AND stock       >= $1
			AND reserved_stock >= $1
	`

	tag, err := exec.Exec(ctx, query,
		qty,
		productID,
		shopID,
	)
	if err != nil {
		return fmt.Errorf("commit inventory failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return inventorydomain.ErrInsufficientReserved
	}

	return nil
}

func (r *InventoryRepository) Restock(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
	qty int,
) error {
	query := `
		UPDATE inventory
		SET
			stock      = stock + $1,
			updated_at = NOW()
		WHERE
			product_id  = $2
			AND shop_id = $3
	`

	tag, err := exec.Exec(ctx, query,
		qty,
		productID,
		shopID,
	)
	if err != nil {
		return fmt.Errorf("restock inventory failed: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("inventory record not found for product %s and shop %s", productID, shopID)
	}

	return nil
}

func (r *InventoryRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	inventory *inventorydomain.Inventory,
) error {
	query := `
		UPDATE inventory
		SET
			stock          = $1,
			reserved_stock = $2,
			updated_at     = NOW()
		WHERE
			id = $3
	`
	_, err := exec.Exec(ctx, query,
		inventory.TotalStock,
		inventory.ReservedStock,
		inventory.ID,
	)
	if err != nil {
		return fmt.Errorf("update inventory failed: %w", err)
	}
	return nil
}

func (r *InventoryRepository) Delete(
	ctx context.Context,
	exec transaction.Executor,
	productID uuid.UUID,
	shopID uuid.UUID,
) error {
	query := `
		DELETE FROM inventory
		WHERE product_id = $1 AND shop_id = $2
	`
	_, err := exec.Exec(ctx, query, productID, shopID)
	if err != nil {
		return fmt.Errorf("delete inventory failed: %w", err)
	}
	return nil
}
