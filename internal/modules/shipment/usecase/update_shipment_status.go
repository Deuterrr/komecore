package usecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	"komecore/internal/modules/shipment/domain"
	"komecore/internal/modules/shipment/repository"

	"github.com/google/uuid"
)

type UpdateShipmentStatusUsecase struct {
	executor     transaction.Executor
	transactor   transaction.Transactor
	shipmentRepo repository.ShipmentRepository
	orderRepo    orderRepo.OrderRepository
}

func NewUpdateShipmentStatusUsecase(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo repository.ShipmentRepository,
	orderRepo orderRepo.OrderRepository,
) *UpdateShipmentStatusUsecase {
	return &UpdateShipmentStatusUsecase{
		executor:     executor,
		transactor:   transactor,
		shipmentRepo: shipmentRepo,
		orderRepo:    orderRepo,
	}
}

type UpdateShipmentStatusInput struct {
	ShipmentID  uuid.UUID
	Status      domain.ShipmentStatus
	Description *string
	Location    *string
}

type UpdateShipmentStatusResult struct {
	Shipment *domain.Shipment
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
		return nil, apperrors.NewNotFound("shipment not found")
	}

	if err := shipment.UpdateStatus(input.Status); err != nil {
		return nil, apperrors.NewInvalidInput(err.Error())
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.shipmentRepo.Update(ctx, exec, *shipment); err != nil {
			return fmt.Errorf("failed to update shipment: %w", err)
		}

		if input.Status == domain.ShipmentStatusDelivered {
			order, err := u.orderRepo.GetByID(ctx, exec, shipment.OrderID)
			if err != nil {
				return fmt.Errorf("failed to get parent order: %w", err)
			}
			if order == nil {
				return apperrors.NewNotFound("parent order not found")
			}

			if err := order.UpdateStatus(orderDomain.OrderStatusDelivered); err != nil {
				return apperrors.NewInvalidInput(err.Error())
			}

			if err := u.orderRepo.UpdateStatus(ctx, exec,
				order.ID,
				orderDomain.OrderStatusDelivered,
			); err != nil {
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
