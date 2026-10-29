package shipment

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

// Repository defines persistence operations for order fulfillment consignments,
// tracking numbers, and courier transit lifecycles.
type Repository interface {
	// Resolves a shipment consignment record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Shipment, error)
	// Resolves the primary shipment consignment linked to an order.
	GetByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Shipment, error)

	// Queries all dispatch consignments associated with an order for multi-parcel fulfillment tracking.
	ListByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]Shipment, error)
	// Batch-loads shipments across multiple orders to eliminate N+1 roundtrips.
	ListByOrderIDs(ctx context.Context, exec transaction.Executor, orderIDs []uuid.UUID) ([]Shipment, error)

	// Inserts a new logistics shipment record with assigned courier, tracking number, and initial created status.
	Create(ctx context.Context, exec transaction.Executor, shipment Shipment) error

	// Updates shipment delivery state, waybill tracking code, or courier transit status.
	Update(ctx context.Context, exec transaction.Executor, shipment Shipment) error
}

// MethodRepository defines persistence operations for shipping service levels, courier methods,
// and fulfillment options available during checkout.
type MethodRepository interface {
	// Queries active shipping methods (e.g. instant, same-day, regular, cargo) available for selection.
	ListActive(ctx context.Context, exec transaction.Executor) ([]ShipmentOption, error)

	// Resolves a shipping method configuration by primary identifier.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*ShipmentOption, error)
}
