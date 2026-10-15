package shipmentusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment"

	"github.com/google/uuid"
)

type UpdateShipmentStatusUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	shipmentRepo  shipment.Repository
	orderDelivery OrderDeliveryUpdater
}

func NewUpdateShipmentStatusUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo shipment.Repository,
	orderDelivery OrderDeliveryUpdater,
) *UpdateShipmentStatusUsecase {
	return &UpdateShipmentStatusUsecase{
		executor:      executor,
		transactor:    transactor,
		shipmentRepo:  shipmentRepo,
		orderDelivery: orderDelivery,
	}
}

type UpdateShipmentStatusInput struct {
	ShipmentID  uuid.UUID
	Status      shipment.ShipmentStatus
	Description *string
	Location    *string
}

type UpdateShipmentStatusResult struct {
	Shipment *shipment.Shipment
}

func (u *UpdateShipmentStatusUsecase) Execute(
	ctx context.Context,
	input UpdateShipmentStatusInput,
) (*UpdateShipmentStatusResult, error) {
	ship, err := u.shipmentRepo.GetByID(ctx, u.executor, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if ship == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	if err := ship.UpdateStatus(input.Status); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.shipmentRepo.Update(ctx, exec, *ship); err != nil {
			return fmt.Errorf("failed to update shipment: %w", err)
		}

		if input.Status == shipment.ShipmentStatusDelivered {
			if err := u.orderDelivery.MarkOrderDelivered(ctx, exec, ship.OrderID); err != nil {
				return fmt.Errorf("failed to update order status to delivered: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	res := UpdateShipmentStatusResult{
		Shipment: ship,
	}

	return &res, nil
}
