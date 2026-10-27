package order

import (
	"fmt"
	"slices"
	"strings"
	"time"

	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
)

// DefaultHandlingSLAWindow defines the maximum duration (72 hours) a merchant has
// to fulfill and ship a confirmed order before it is automatically cancelled by the expiry worker.
const DefaultHandlingSLAWindow = 72 * time.Hour

// OrderStatus represents the discrete lifecycle states of an Order aggregate in the domain state machine.
type OrderStatus string

const (
	// OrderStatusPending indicates the order is created and awaiting payment gateway authorization.
	// Inventory is reserved with temporary holds; unpaids expire after the payment TTL (24h).
	OrderStatusPending OrderStatus = "pending"

	// OrderStatusConfirmed indicates payment is settled and verified.
	// Inventory is committed to the order, and the 72-hour merchant handling SLA begins.
	OrderStatusConfirmed OrderStatus = "confirmed"

	// OrderStatusProcessing indicates the merchant has acknowledged the order and is preparing items for dispatch.
	// Order cancellation from this state requires automated customer refund and inventory restocking.
	OrderStatusProcessing OrderStatus = "processing"

	// OrderStatusShipped indicates all items have been dispatched to couriers with active tracking numbers.
	// The order can no longer be cancelled or refunded through standard cancellation workflows.
	OrderStatusShipped OrderStatus = "shipped"

	// OrderStatusDelivered is the successful terminal state indicating the customer received all shipments.
	OrderStatusDelivered OrderStatus = "delivered"

	// OrderStatusCancelled is an aborted terminal state triggered by customer or staff before shipment.
	// Releases uncommitted holds or restocks committed inventory and initiates payment refund.
	OrderStatusCancelled OrderStatus = "cancelled"

	// OrderStatusExpired is an aborted terminal state triggered by background workers when either
	// the payment window lapses (pending) or merchant fulfillment SLA breaches (confirmed/processing).
	OrderStatusExpired OrderStatus = "expired"
)

// allowedTransitions defines permitted state transitions; empty sets enforce lifecycle finality.
var allowedTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending: {
		OrderStatusConfirmed, // Payment settled by gateway webhook
		OrderStatusCancelled, // Customer or staff aborted before payment
		OrderStatusExpired,   // Payment TTL window (24h) timed out
	},

	OrderStatusConfirmed: {
		OrderStatusProcessing, // Merchant acknowledges and starts packing
		OrderStatusCancelled,  // Staff cancels order (triggers refund + restock)
		OrderStatusExpired,    // 72h handling SLA breached without fulfillment
	},

	OrderStatusProcessing: {
		OrderStatusShipped,   // All packages dispatched with active tracking
		OrderStatusCancelled, // Staff cancels order (triggers refund + restock)
		OrderStatusExpired,   // 72h handling SLA breached without fulfillment
	},

	OrderStatusShipped: {
		OrderStatusDelivered, // Customer receives delivery; courier custody complete
	},

	OrderStatusDelivered: {}, // Terminal: order successfully fulfilled
	OrderStatusCancelled: {}, // Terminal: order aborted prior to shipment
	OrderStatusExpired:   {}, // Terminal: payment or fulfillment SLA lapsed
}

// Order is the aggregate root capturing multi-shop checkout contracts, financial totals,
// and fulfillment SLA timestamps. Any mutation must preserve allowedTransitions and SLA invariants.
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

// IsHandlingExpired evaluates whether the merchant has breached the 72-hour handling window.
// Returns false for orders without an established SLA timestamp.
func (o Order) IsHandlingExpired(now time.Time) bool {
	if o.HandlingExpiresAt == nil {
		return false
	}
	return !now.Before(*o.HandlingExpiresAt)
}

// Confirm advances the order from Pending to Confirmed upon payment settlement.
// It establishes the 72-hour handling SLA deadline (HandlingExpiresAt) required for downstream expiry checks.
func (o *Order) Confirm(confirmedAt time.Time, slaWindow time.Duration) error {
	if slaWindow <= 0 {
		slaWindow = DefaultHandlingSLAWindow
	}
	if !o.canTransitionTo(OrderStatusConfirmed) && o.Status != OrderStatusConfirmed {
		return fmt.Errorf("invalid status transition: %s → %s", o.Status, OrderStatusConfirmed)
	}

	t := confirmedAt.UTC()
	expiresAt := t.Add(slaWindow)

	o.Status = OrderStatusConfirmed
	o.ConfirmedAt = &t
	o.HandlingExpiresAt = &expiresAt

	return o.Validate()
}

// Validate enforces domain integrity rules for the Order aggregate.
// Orders in Confirmed or Processing status MUST have ConfirmedAt and HandlingExpiresAt stamped.
func (o Order) Validate() error {
	if o.Status == OrderStatusConfirmed || o.Status == OrderStatusProcessing {
		if o.ConfirmedAt == nil || o.HandlingExpiresAt == nil {
			return ErrMissingSLAFields
		}
	}

	return nil
}

// UpdateStatus mutates order status adhering strictly to the FSM allowedTransitions matrix.
// If transitioning to Confirmed or Processing without SLA stamps, it initializes them using now.
func (o *Order) UpdateStatus(status OrderStatus) error {
	if o.Status == status {
		return o.Validate()
	}
	if !o.canTransitionTo(status) {
		return fmt.Errorf("invalid status transition: %s → %s", o.Status, status)
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

// NewInvoice generates an immutable financial invoice snapshot mirroring order totals
// for accounting, customer tax receipts, and payment gateway reconciliation.
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
