package domain

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type (
	// ShipmentStatus represents the operational status of a
	// shipment throughout the delivery process.
	ShipmentStatus string
	// FulfillmentMethod identifies how an order is fulfilled
	// and delivered to the customer.
	FulfillmentMethod string
)

const (
	ShipmentStatusCreated        ShipmentStatus = "created"
	ShipmentStatusPacked         ShipmentStatus = "packed"
	ShipmentStatusLabelled       ShipmentStatus = "labelled"
	ShipmentStatusPickedUp       ShipmentStatus = "picked_up"
	ShipmentStatusInTransit      ShipmentStatus = "in_transit"
	ShipmentStatusShipped        ShipmentStatus = "shipped"
	ShipmentStatusOutForDelivery ShipmentStatus = "out_for_delivery"
	ShipmentStatusDelivered      ShipmentStatus = "delivered"
	ShipmentStatusFailed         ShipmentStatus = "failed"
	ShipmentStatusReturned       ShipmentStatus = "returned"
	ShipmentStatusCancelled      ShipmentStatus = "cancelled"

	FulfillmentMethodCourier      FulfillmentMethod = "courier"
	FulfillmentMethodSelfDelivery FulfillmentMethod = "self_delivery"
)

var allowedShipmentTransitions = map[ShipmentStatus][]ShipmentStatus{
	ShipmentStatusCreated: {
		ShipmentStatusPacked,
		ShipmentStatusLabelled,
		ShipmentStatusPickedUp,
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusCancelled,
	},
	ShipmentStatusPacked: {
		ShipmentStatusLabelled,
		ShipmentStatusPickedUp,
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusCancelled,
	},
	ShipmentStatusLabelled: {
		ShipmentStatusPickedUp,
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusCancelled,
	},
	ShipmentStatusPickedUp: {
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusFailed,
		ShipmentStatusCancelled,
	},
	ShipmentStatusInTransit: {
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusFailed,
		ShipmentStatusCancelled,
	},
	ShipmentStatusShipped: {
		ShipmentStatusInTransit,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusFailed,
		ShipmentStatusCancelled,
	},
	ShipmentStatusOutForDelivery: {
		ShipmentStatusDelivered,
		ShipmentStatusFailed,
		ShipmentStatusCancelled,
	},
	ShipmentStatusFailed: {
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusReturned,
		ShipmentStatusCancelled,
	},
	ShipmentStatusDelivered: {},
	ShipmentStatusReturned:  {},
	ShipmentStatusCancelled: {},
}

type Shipment struct {
	ID uuid.UUID

	OrderID uuid.UUID

	Status            ShipmentStatus
	FulfillmentMethod FulfillmentMethod
	TrackingNumber    *string

	Courier string
	Service string

	Cost   int64
	Weight int

	ShippedAt   *time.Time
	DeliveredAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

func (d *Shipment) Validate() error {
	if d.FulfillmentMethod == "" {
		d.FulfillmentMethod = FulfillmentMethodCourier
	}

	if d.FulfillmentMethod != FulfillmentMethodCourier && d.FulfillmentMethod != FulfillmentMethodSelfDelivery {
		return ErrInvalidFulfillmentMethod
	}

	if d.Cost < 0 {
		return ErrInvalidCost
	}

	if d.Weight <= 0 {
		return ErrInvalidWeight
	}

	if d.FulfillmentMethod == FulfillmentMethodCourier {
		if d.Courier == "" {
			return ErrInvalidCourier
		}

		if d.Service == "" {
			return ErrInvalidService
		}
	}

	return nil
}

func (d *Shipment) UpdateStatus(status ShipmentStatus) error {
	if d.Status == status {
		return nil
	}

	if !d.canTransitionTo(status) {
		return fmt.Errorf("invalid shipment status transition: %s → %s", d.Status, status)
	}

	d.Status = status
	now := time.Now()
	d.UpdatedAt = &now

	if status == ShipmentStatusShipped || status == ShipmentStatusInTransit {
		if d.ShippedAt == nil {
			d.ShippedAt = &now
		}
	} else if status == ShipmentStatusDelivered {
		if d.DeliveredAt == nil {
			d.DeliveredAt = &now
		}
	}

	return nil
}

func (d *Shipment) canTransitionTo(next ShipmentStatus) bool {
	allowed, exists := allowedShipmentTransitions[d.Status]
	if !exists {
		return false
	}

	return slices.Contains(allowed, next)
}

func (s ShipmentStatus) IsValid() bool {
	switch s {
	case
		ShipmentStatusCreated,
		ShipmentStatusPacked,
		ShipmentStatusLabelled,
		ShipmentStatusPickedUp,
		ShipmentStatusInTransit,
		ShipmentStatusShipped,
		ShipmentStatusOutForDelivery,
		ShipmentStatusDelivered,
		ShipmentStatusFailed,
		ShipmentStatusReturned,
		ShipmentStatusCancelled:

		return true
	default:
		return false
	}
}
