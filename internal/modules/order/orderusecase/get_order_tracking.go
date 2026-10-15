package orderusecase

import (
	"context"
	"fmt"
	"time"

	"komecore/internal/apperror"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order"
	"komecore/internal/modules/shipment"

	"github.com/google/uuid"
)

type GetOrderTrackingUsecase struct {
	executor     transaction.Executor
	orderRepo    order.Repository
	shipmentRepo shipment.Repository
}

func NewGetOrderTrackingUsecase(
	executor transaction.Executor,
	orderRepo order.Repository,
	shipmentRepo shipment.Repository,
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
	ord, err := u.orderRepo.GetByID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	if ord == nil {
		return nil, apperror.NewNotFoundWithCode(apperror.CodeOrderNotFound, "order not found")
	}

	if input.CustomerID != uuid.Nil && ord.CustomerID != input.CustomerID {
		return nil, apperror.NewUnauthorized("not authorized")
	}

	var shp *shipment.Shipment
	if input.ShipmentID != nil {
		s, err := u.shipmentRepo.GetByID(ctx, u.executor, *input.ShipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shipment: %w", err)
		}
		if s != nil && s.OrderID == ord.ID {
			shp = s
		}
	}

	if shp == nil {
		shipments, err := u.shipmentRepo.ListByOrderID(ctx, u.executor, ord.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list shipments: %w", err)
		}
		if len(shipments) > 0 {
			shp = &shipments[0]
		}
	}

	if shp == nil {
		return nil, apperror.NewNotFound("shipment not found")
	}

	var timeline []TrackingTimelineEvent
	timeline = append(timeline, TrackingTimelineEvent{
		Status:      string(shipment.ShipmentStatusCreated),
		Description: "Shipment created",
		Location:    "",
		Timestamp:   shp.CreatedAt,
	})

	if shp.ShippedAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipment.ShipmentStatusShipped),
			Description: fmt.Sprintf("Shipped via %s (%s)", shp.Courier, shp.Service),
			Location:    "",
			Timestamp:   *shp.ShippedAt,
		})
	}
	if shp.DeliveredAt != nil {
		timeline = append(timeline, TrackingTimelineEvent{
			Status:      string(shipment.ShipmentStatusDelivered),
			Description: "Shipment delivered",
			Location:    "",
			Timestamp:   *shp.DeliveredAt,
		})
	}

	return &GetOrderTrackingResult{
		OrderID:        ord.ID,
		ShipmentID:     shp.ID,
		Courier:        shp.Courier,
		TrackingNumber: shp.TrackingNumber,
		Timeline:       timeline,
	}, nil
}
