package shipmentusecase

import (
	"context"
	"fmt"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shipment/shipmentrepo"

	"github.com/google/uuid"
)

type UpdateShipmentStatusUsecase struct {
	executor      transaction.Executor
	transactor    transaction.Transactor
	shipmentRepo  shipmentrepo.ShipmentRepository
	orderDelivery OrderDeliveryUpdater
}

func NewUpdateShipmentStatusUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo shipmentrepo.ShipmentRepository,
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
	Status      shipmentdomain.ShipmentStatus
	Description *string
	Location    *string
}

type UpdateShipmentStatusResult struct {
	Shipment *shipmentdomain.Shipment
}

func (u *UpdateShipmentStatusUsecase) Execute(
	ctx context.Context,
	input UpdateShipmentStatusInput,
) (*UpdateShipmentStatusResult, error) {
	shipment, err := u.shipmentRepo.GetByID(ctx, u.executor, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if shipment == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	if err := shipment.UpdateStatus(input.Status); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.shipmentRepo.Update(ctx, exec, *shipment); err != nil {
			return fmt.Errorf("failed to update shipment: %w", err)
		}

		if input.Status == shipmentdomain.ShipmentStatusDelivered {
			if err := u.orderDelivery.MarkOrderDelivered(ctx, exec, shipment.OrderID); err != nil {
				return fmt.Errorf("failed to update order status to delivered: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	res := UpdateShipmentStatusResult{
		Shipment: shipment,
	}

	return &res, nil
}
