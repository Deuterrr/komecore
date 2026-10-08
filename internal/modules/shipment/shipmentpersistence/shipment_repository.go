package shipmentpersistence

import (
	"context"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/shipmentdomain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ShipmentRepository struct{}

func NewShipmentRepository() *ShipmentRepository {
	return &ShipmentRepository{}
}

const shipmentSelectCols = `
	id,
	order_id,
	status,
	fulfillment_method,
	tracking_number,
	courier_name,
	service,
	shipping_cost,
	weight,
	origin_id,
	destination_id,
	created_at
`

func (r *ShipmentRepository) scanShipment(row transaction.Row) (*shipmentdomain.Shipment, error) {
	var (
		s                shipmentdomain.Shipment
		originID, destID string
	)
	err := row.Scan(
		&s.ID,
		&s.OrderID,
		&s.Status,
		&s.FulfillmentMethod,
		&s.TrackingNumber,
		&s.Courier,
		&s.Service,
		&s.Cost,
		&s.Weight,
		&originID,
		&destID,
		&s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan shipment failed: %w", err)
	}
	return &s, nil
}

func (r *ShipmentRepository) GetByID(
	ctx context.Context,
	exec transaction.Executor,
	id uuid.UUID,
) (*shipmentdomain.Shipment, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM shipments
		WHERE id = $1
	`, shipmentSelectCols)

	s, err := r.scanShipment(exec.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query shipment by id failed: %w", err)
	}
	return s, nil
}

func (r *ShipmentRepository) GetByOrderID(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) (*shipmentdomain.Shipment, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM shipments
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, shipmentSelectCols)

	s, err := r.scanShipment(exec.QueryRow(ctx, query, orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query shipment by order id failed: %w", err)
	}
	return s, nil
}

func (r *ShipmentRepository) ListByOrderID(
	ctx context.Context,
	exec transaction.Executor,
	orderID uuid.UUID,
) ([]shipmentdomain.Shipment, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM shipments
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, shipmentSelectCols)

	rows, err := exec.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query shipments by order id failed: %w", err)
	}
	defer rows.Close()

	shipments, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (shipmentdomain.Shipment, error) {
		var (
			s                shipmentdomain.Shipment
			originID, destID string
		)
		err := row.Scan(
			&s.ID,
			&s.OrderID,
			&s.Status,
			&s.FulfillmentMethod,
			&s.TrackingNumber,
			&s.Courier,
			&s.Service,
			&s.Cost,
			&s.Weight,
			&originID,
			&destID,
			&s.CreatedAt,
		)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan shipments failed: %w", err)
	}

	return shipments, nil
}

func (r *ShipmentRepository) ListByOrderIDs(
	ctx context.Context,
	exec transaction.Executor,
	orderIDs []uuid.UUID,
) ([]shipmentdomain.Shipment, error) {
	if len(orderIDs) == 0 {
		return []shipmentdomain.Shipment{}, nil
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM shipments
		WHERE order_id = ANY($1::uuid[])
		ORDER BY created_at ASC
	`, shipmentSelectCols)

	orderIDStrings := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		orderIDStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, orderIDStrings)
	if err != nil {
		return nil, fmt.Errorf("query shipments by order ids failed: %w", err)
	}
	defer rows.Close()

	shipments, err := transaction.CollectRows(rows, func(row transaction.CollectableRow) (shipmentdomain.Shipment, error) {
		var (
			s                shipmentdomain.Shipment
			originID, destID string
		)
		err := row.Scan(
			&s.ID,
			&s.OrderID,
			&s.Status,
			&s.FulfillmentMethod,
			&s.TrackingNumber,
			&s.Courier,
			&s.Service,
			&s.Cost,
			&s.Weight,
			&originID,
			&destID,
			&s.CreatedAt,
		)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan shipments failed: %w", err)
	}

	return shipments, nil
}

func (r *ShipmentRepository) Create(
	ctx context.Context,
	exec transaction.Executor,
	shipment shipmentdomain.Shipment,
) error {
	query := `
		INSERT INTO shipments (
			id,
			order_id,
			status,
			fulfillment_method,
			tracking_number,
			courier_name,
			service,
			shipping_cost,
			weight,
			origin_id,
			destination_id,
			created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`

	_, err := exec.Exec(ctx, query,
		shipment.ID,
		shipment.OrderID,
		shipment.Status,
		shipment.FulfillmentMethod,
		shipment.TrackingNumber,
		shipment.Courier,
		shipment.Service,
		shipment.Cost,
		shipment.Weight,
		"",
		"",
		shipment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create shipment failed: %w", err)
	}
	return nil
}

func (r *ShipmentRepository) Update(
	ctx context.Context,
	exec transaction.Executor,
	shipment shipmentdomain.Shipment,
) error {
	query := `
		UPDATE shipments
		SET
			status             = $2,
			fulfillment_method = $3,
			tracking_number    = $4,
			courier_name       = $5,
			service            = $6,
			shipping_cost      = $7,
			weight             = $8
		WHERE id = $1
	`

	_, err := exec.Exec(ctx, query,
		shipment.ID,
		shipment.Status,
		shipment.FulfillmentMethod,
		shipment.TrackingNumber,
		shipment.Courier,
		shipment.Service,
		shipment.Cost,
		shipment.Weight,
	)
	if err != nil {
		return fmt.Errorf("update shipment failed: %w", err)
	}
	return nil
}
