package repository

import (
	"context"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/domain"

	"github.com/google/uuid"
)

type ShipmentRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.Shipment, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*domain.Shipment, error)

	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]domain.Shipment, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]domain.Shipment, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		shipment domain.Shipment,
	) error
	Update(
		ctx context.Context,
		exec transaction.Executor,
		shipment domain.Shipment,
	) error
}

type ShipmentMethodRepository interface {
	ListActive(
		ctx context.Context,
		exec transaction.Executor,
	) ([]domain.ShipmentOption, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*domain.ShipmentOption, error)
}
