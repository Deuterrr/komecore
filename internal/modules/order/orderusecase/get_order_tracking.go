package orderusecase

import (
	"context"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderrepo"
	"komecore/internal/modules/shipment/shipmentdomain"
	"komecore/internal/modules/shipment/shipmentrepo"

	"github.com/google/uuid"
)

type GetOrderTrackingUsecase struct {
	executor     transaction.Executor
	orderRepo    orderrepo.OrderRepository
	shipmentRepo shipmentrepo.ShipmentRepository
}

func NewGetOrderTrackingUsecase(
	executor transaction.Executor,
	orderRepo orderrepo.OrderRepository,
	shipmentRepo shipmentrepo.ShipmentRepository,
) *GetOrderTrackingUsecase {
	return &GetOrderTrackingUsecase{
		executor:     executor,
		orderRepo:    orderRepo,
		shipmentRepo: shipmentRepo,
	}
}

type GetOrderTrackingInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
	ShipmentID *uuid.UUID
}

type TrackingTimelineEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Timestamp   time.Time `json:"timestamp"`
}

type GetOrderTrackingResult struct {
	OrderID        uuid.UUID               `json:"order_id"`
	ShipmentID     uuid.UUID               `json:"shipment_id"`
	Courier        string                  `json:"courier"`
	TrackingNumber *string                 `json:"tracking_number,omitempty"`
	Warning        *string                 `json:"warning,omitempty"`
	Timeline       []TrackingTimelineEvent `json:"timeline"`
}

func (u *GetOrderTrackingUsecase) Execute(
	ctx context.Context,
	input GetOrderTrackingInput,
) (*GetOrderTrackingResult, error) {
	order, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return nil, apperrors.NewNotFound("order not found")
	}

	if input.CustomerID != uuid.Nil && order.CustomerID != input.CustomerID {
		return nil, apperrors.NewUnauthorized("not authorized")
	}

	var shipment *shipmentdomain.Shipment
	if input.ShipmentID != nil {
		s, err := u.shipmentRepo.GetByID(ctx, u.executor, *input.ShipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shipment: %w", err)
		}
		if s != nil && s.OrderID == order.ID {
			shipment = s
		}
	}

	if shipment == nil {
		shipments, err := u.shipmentRepo.ListByOrderID(ctx, u.executor, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list shipments: %w", err)
		}
		if len(shipments) > 0 {
			shipment = &shipments[0]
		}
	}

	if shipment == nil {
		return nil, apperrors.NewNotFound("shipment not found")
	}

	var timeline []TrackingTimelineEvent
	timeline = append(timeline, TrackingTimelineEvent{
		Status:      string(shipmentdomain.ShipmentStatusCreated),
		Description: "Shipment created",
		Location:    "",
		Timestamp:   shipment.CreatedAt,
	})

	if shipment.ShippedAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipmentdomain.ShipmentStatusShipped),
			Description: fmt.Sprintf("Shipped via %s (%s)", shipment.Courier, shipment.Service),
			Location:    "",
			Timestamp:   *shipment.ShippedAt,
		})
	}
	if shipment.DeliveredAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipmentdomain.ShipmentStatusDelivered),
			Description: "Shipment delivered",
			Location:    "",
			Timestamp:   *shipment.DeliveredAt,
		})
	}

	return &GetOrderTrackingResult{
		OrderID:        order.ID,
		ShipmentID:     shipment.ID,
		Courier:        shipment.Courier,
		TrackingNumber: shipment.TrackingNumber,
		Timeline:       timeline,
	}, nil
}
