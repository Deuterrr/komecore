package shipmentrepo

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/shipmentdomain"

	"github.com/google/uuid"
)

type ShipmentRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*shipmentdomain.Shipment, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*shipmentdomain.Shipment, error)

	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]shipmentdomain.Shipment, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]shipmentdomain.Shipment, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		shipment shipmentdomain.Shipment,
	) error
	Update(
		ctx context.Context,
		exec transaction.Executor,
		shipment shipmentdomain.Shipment,
	) error
}

type ShipmentMethodRepository interface {
	ListActive(
		ctx context.Context,
		exec transaction.Executor,
	) ([]shipmentdomain.ShipmentOption, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*shipmentdomain.ShipmentOption, error)
}
