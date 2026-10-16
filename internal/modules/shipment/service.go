package shipment

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	shipping "komecore/internal/infra/shipping"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/courier"

	"github.com/google/uuid"
)

type OrderDeliveryUpdater interface {
	MarkOrderDelivered(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) error
}

type Service struct {
	executor         transaction.Executor
	transactor       transaction.Transactor
	shipmentRepo     Repository
	courierRepo      courier.Repository
	shippingProvider shipping.ShippingProvider
	orderDelivery    OrderDeliveryUpdater
}

func NewService(
	executor transaction.Executor,
	transactor transaction.Transactor,
	shipmentRepo Repository,
	courierRepo courier.Repository,
	shippingProvider shipping.ShippingProvider,
	orderDelivery OrderDeliveryUpdater,
) *Service {
	return &Service{
		executor:         executor,
		transactor:       transactor,
		shipmentRepo:     shipmentRepo,
		courierRepo:      courierRepo,
		shippingProvider: shippingProvider,
		orderDelivery:    orderDelivery,
	}
}

type EstimateShippingOptionsInput struct {
	ShopID      uuid.UUID
	Origin      int
	Destination int
	Weight      int
	PriceFilter *string
}

func (s *Service) EstimateShippingOptions(ctx context.Context, input EstimateShippingOptionsInput) ([]shipping.RateOption, error) {
	if input.Weight <= 0 {
		return nil, apperror.NewInvalidInput(ErrInvalidWeight.Error())
	}

	var courierCodes []string
	if s.courierRepo != nil {
		codes, err := s.courierRepo.ListAll(ctx, s.executor)
		if err == nil && len(codes) > 0 {
			courierCodes = codes
		}
	}
	if len(courierCodes) == 0 {
		courierCodes = []string{"jne", "jnt", "sicepat", "flat"}
	}

	query := shipping.CalculateRatesInput{
		OriginID:      input.Origin,
		DestinationID: input.Destination,
		Weight:        input.Weight,
		Couriers:      courierCodes,
		PriceFilter:   input.PriceFilter,
	}
	costOptions, err := s.shippingProvider.CalculateRates(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate shipping cost: %w", err)
	}

	return costOptions, nil
}

type UpdateShipmentInput struct {
	ShipmentID     uuid.UUID
	TrackingNumber *string
	Courier        *string
	Service        *string
}

type UpdateShipmentResult struct {
	Shipment *Shipment
}

func (s *Service) UpdateShipment(ctx context.Context, input UpdateShipmentInput) (*UpdateShipmentResult, error) {
	ship, err := s.shipmentRepo.GetByID(ctx, s.executor, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if ship == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	updated := false
	if input.TrackingNumber != nil {
		ship.TrackingNumber = input.TrackingNumber
		updated = true
	}
	if input.Courier != nil {
		ship.Courier = *input.Courier
		updated = true
	}
	if input.Service != nil {
		ship.Service = *input.Service
		updated = true
	}

	if !updated {
		return &UpdateShipmentResult{
			Shipment: ship,
		}, nil
	}

	if err := ship.Validate(); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		return s.shipmentRepo.Update(ctx, exec, *ship)
	})
	if err != nil {
		return nil, err
	}

	res := UpdateShipmentResult{
		Shipment: ship,
	}
	return &res, nil
}

func (s *Service) Dispatch(ctx context.Context, shipmentID uuid.UUID, trackingNumber string) (*Shipment, error) {
	ship, err := s.shipmentRepo.GetByID(ctx, s.executor, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if ship == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	ship.TrackingNumber = &trackingNumber
	now := time.Now()
	ship.ShippedAt = &now
	if err := ship.UpdateStatus(ShipmentStatusShipped); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		return s.shipmentRepo.Update(ctx, exec, *ship)
	})
	if err != nil {
		return nil, err
	}

	return ship, nil
}

type UpdateShipmentStatusInput struct {
	ShipmentID  uuid.UUID
	Status      ShipmentStatus
	Description *string
	Location    *string
}

type UpdateShipmentStatusResult struct {
	Shipment *Shipment
}

func (s *Service) UpdateShipmentStatus(ctx context.Context, input UpdateShipmentStatusInput) (*UpdateShipmentStatusResult, error) {
	ship, err := s.shipmentRepo.GetByID(ctx, s.executor, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}
	if ship == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	if err := ship.UpdateStatus(input.Status); err != nil {
		return nil, apperror.NewInvalidInput(err.Error())
	}

	err = s.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := s.shipmentRepo.Update(ctx, exec, *ship); err != nil {
			return fmt.Errorf("failed to update shipment: %w", err)
		}

		if input.Status == ShipmentStatusDelivered {
			if err := s.orderDelivery.MarkOrderDelivered(ctx, exec, ship.OrderID); err != nil {
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
