package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

const (
	EventOrderCreated       = "order.created"
	EventPaymentSettled     = "payment.settled"
	EventShipmentDispatched = "shipment.dispatched"
)

type Event struct {
	ID          uuid.UUID       `json:"id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	Status      Status          `json:"status"`
	RetryCount  int             `json:"retry_count"`
	MaxRetries  int             `json:"max_retries"`
	LastError   *string         `json:"last_error"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	ProcessedAt *time.Time      `json:"processed_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

type OrderCreatedItem struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Subtotal    int64  `json:"subtotal"`
}

type OrderCreatedPayload struct {
	OrderID       uuid.UUID          `json:"order_id"`
	OrderNumber   string             `json:"order_number"`
	CustomerEmail string             `json:"customer_email"`
	CustomerName  string             `json:"customer_name"`
	Total         int64              `json:"total"`
	Items         []OrderCreatedItem `json:"items"`
}

type PaymentSettledPayload struct {
	OrderID       uuid.UUID `json:"order_id"`
	OrderNumber   string    `json:"order_number"`
	CustomerEmail string    `json:"customer_email"`
	CustomerName  string    `json:"customer_name"`
	Amount        int64     `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	PaidAt        time.Time `json:"paid_at"`
}

type ShipmentDispatchedPayload struct {
	OrderID        uuid.UUID `json:"order_id"`
	OrderNumber    string    `json:"order_number"`
	CustomerEmail  string    `json:"customer_email"`
	CustomerName   string    `json:"customer_name"`
	TrackingNumber string    `json:"tracking_number"`
	Courier        string    `json:"courier"`
	Service        string    `json:"service"`
}
