package usecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/shipment/domain"
	"komecore/internal/modules/shipment/repository"

	"github.com/google/uuid"
)

type UpdateShipmentUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	shipmentRepo repository.ShipmentRepository
}

func NewUpdateShipmentUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo repository.ShipmentRepository,
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
	Shipment *domain.Shipment
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
		return nil, apperrors.NewNotFound("shipment not found")
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
		return nil, apperrors.NewInvalidInput(err.Error())
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
) (*domain.Shipment, error) {
	shipment, err := u.shipmentRepo.GetByID(ctx, u.executor, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if shipment == nil {
		return nil, apperrors.NewNotFound("shipment not found")
	}

	shipment.TrackingNumber = &trackingNumber
	now := time.Now()
	shipment.ShippedAt = &now
	if err := shipment.UpdateStatus(domain.ShipmentStatusShipped); err != nil {
		return nil, apperrors.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		return u.shipmentRepo.Update(ctx, exec, *shipment)
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}
