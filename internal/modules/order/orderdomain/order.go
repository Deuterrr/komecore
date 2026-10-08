package orderdomain

import (
	"fmt"
	"slices"
	"strings"
	"time"

	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

const DefaultHandlingSLAWindow = 72 * time.Hour

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusExpired    OrderStatus = "expired"
)

var allowedTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending: {
		OrderStatusConfirmed,
		OrderStatusCancelled,
		OrderStatusExpired,
	},

	OrderStatusConfirmed: {
		OrderStatusProcessing,
		OrderStatusCancelled,
		OrderStatusExpired,
	},

	OrderStatusProcessing: {
		OrderStatusShipped,
		OrderStatusCancelled,
		OrderStatusExpired,
	},

	OrderStatusShipped: {
		OrderStatusDelivered,
	},

	OrderStatusDelivered: {},

	OrderStatusCancelled: {},

	OrderStatusExpired: {},
}

type Order struct {
	ID     uuid.UUID
	Number string

	CustomerID uuid.UUID
	AddressID  uuid.UUID

	Status OrderStatus

	Subtotal    int64
	ShippingFee int64
	Total       int64

	ConfirmedAt       *time.Time
	HandlingExpiresAt *time.Time

	CreatedAt time.Time
	UpdatedAt *time.Time
}

func (o Order) IsHandlingExpired(now time.Time) bool {
	if o.HandlingExpiresAt == nil {
		return false
	}
	return !now.Before(*o.HandlingExpiresAt)
}

// Confirm transitions the order to confirmed status and stamps SLA fields.
func (o *Order) Confirm(confirmedAt time.Time, slaWindow time.Duration) error {
	if slaWindow <= 0 {
		slaWindow = DefaultHandlingSLAWindow
	}
	if !o.canTransitionTo(OrderStatusConfirmed) && o.Status != OrderStatusConfirmed {
		return fmt.Errorf("invalid status transition: %s â†’ %s", o.Status, OrderStatusConfirmed)
	}

	t := confirmedAt.UTC()
	expiresAt := t.Add(slaWindow)

	o.Status = OrderStatusConfirmed
	o.ConfirmedAt = &t
	o.HandlingExpiresAt = &expiresAt

	return o.Validate()
}

// Validate checks domain invariants for Order aggregate.
func (o Order) Validate() error {
	if o.Status == OrderStatusConfirmed || o.Status == OrderStatusProcessing {
		if o.ConfirmedAt == nil || o.HandlingExpiresAt == nil {
			return ErrMissingSLAFields
		}
	}

	return nil
}

func (o *Order) UpdateStatus(status OrderStatus) error {
	if o.Status == status {
		return o.Validate()
	}

	if !o.canTransitionTo(status) {
		return fmt.Errorf("invalid status transition: %s â†’ %s", o.Status, status)
	}

	o.Status = status
	if (status == OrderStatusConfirmed || status == OrderStatusProcessing) &&
		o.ConfirmedAt == nil {

		now := appclock.Now()
		expiresAt := now.Add(DefaultHandlingSLAWindow)

		o.ConfirmedAt = &now
		o.HandlingExpiresAt = &expiresAt
	}

	return o.Validate()
}

func (o Order) NewInvoice() Invoice {
	return Invoice{
		ID:          uuid.New(),
		Number:      NewInvoiceNumber(),
		OrderID:     o.ID,
		Status:      InvoiceStatusIssued,
		Subtotal:    o.Subtotal,
		ShippingFee: o.ShippingFee,
		Total:       o.Total,
		IssuedAt:    o.CreatedAt,
		CreatedAt:   o.CreatedAt,
	}
}

func (o *Order) canTransitionTo(next OrderStatus) bool {
	allowed, exists := allowedTransitions[o.Status]
	if !exists {
		return false
	}

	return slices.Contains(allowed, next)
}

func NewOrderNumber() string {
	return fmt.Sprintf(
		"ORD-%s-%s",
		appclock.Now().Format("20060102"),
		strings.ToUpper(uuid.NewString()[:6]),
	)
}
