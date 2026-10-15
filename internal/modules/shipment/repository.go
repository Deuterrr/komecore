package shipment

import (
	"context"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
)

type Repository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Shipment, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*Shipment, error)

	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]Shipment, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]Shipment, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		shipment Shipment,
	) error

	Update(
		ctx context.Context,
		exec transaction.Executor,
		shipment Shipment,
	) error
}

type MethodRepository interface {
	ListActive(
		ctx context.Context,
		exec transaction.Executor,
	) ([]ShipmentOption, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*ShipmentOption, error)
}

type ShipmentRepository = Repository
type ShipmentMethodRepository = MethodRepository
