package shipmentusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shipment/shipmentrepo"

	"github.com/google/uuid"
)

type UpdateShipmentUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	shipmentRepo shipmentrepo.ShipmentRepository
}

func NewUpdateShipmentUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo shipmentrepo.ShipmentRepository,
) *UpdateShipmentUsecase {
	return &UpdateShipmentUsecase{
		executor:     executor,
		transactor:   transactor,
		shipmentRepo: shipmentRepo,
	}
}

type UpdateShipmentInput struct {
	ShipmentID     uuid.UUID
	TrackingNumber *string
	Courier        *string
	Service        *string
}

type UpdateShipmentResult struct {
	Shipment *shipmentdomain.Shipment
}

func (u *UpdateShipmentUsecase) Execute(
	ctx context.Context,
	input UpdateShipmentInput,
) (*UpdateShipmentResult, error) {
	shipment, err := u.shipmentRepo.GetByID(ctx, u.executor, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if shipment == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	updated := false
	if input.TrackingNumber != nil {
		shipment.TrackingNumber = input.TrackingNumber
		updated = true
	}
	if input.Courier != nil {
		shipment.Courier = *input.Courier
		updated = true
	}
	if input.Service != nil {
		shipment.Service = *input.Service
		updated = true
	}

	if !updated {
		return &UpdateShipmentResult{
			Shipment: shipment,
		}, nil
	}

	if err := shipment.Validate(); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		return u.shipmentRepo.Update(ctx, exec, *shipment)
	})
	if err != nil {
		return nil, err
	}

	res := UpdateShipmentResult{
		Shipment: shipment,
	}

	return &res, nil
}

func (u *UpdateShipmentUsecase) Dispatch(
	ctx context.Context,
	shipmentID uuid.UUID,
	trackingNumber string,
) (*shipmentdomain.Shipment, error) {
	shipment, err := u.shipmentRepo.GetByID(ctx, u.executor, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if shipment == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	shipment.TrackingNumber = &trackingNumber
	now := time.Now()
	shipment.ShippedAt = &now
	if err := shipment.UpdateStatus(shipmentdomain.ShipmentStatusShipped); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		return u.shipmentRepo.Update(ctx, exec, *shipment)
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}
