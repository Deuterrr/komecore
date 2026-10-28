package payment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Processing lifecycle states for an inbound gateway notification.
type WebhookEventStatus string

const (
	// Raw payload stored in DB, queued for domain transition processing.
	WebhookEventStatusReceived WebhookEventStatus = "received"
	// Successfully reconciled and applied to the payment domain aggregate.
	WebhookEventStatusProcessed WebhookEventStatus = "processed"
	// Domain processing failed; retained with error message for diagnostics and retry on redelivery.
	WebhookEventStatusFailed WebhookEventStatus = "failed"
)

// Inbound webhook notification persisted prior to domain execution.
//
// Invariant: (OrderID, TransactionStatus) is the unique idempotency barrier preventing duplicate transitions.
type PaymentWebhookEvent struct {
	ID uuid.UUID
	// Merchant or gateway order reference parsed from the webhook body.
	OrderID string

	// Gateway status field forming the unique idempotency key with OrderID.
	TransactionStatus string
	Status            WebhookEventStatus

	// Full raw JSON payload as delivered by the upstream gateway.
	Payload json.RawMessage
	// Error details recorded when Status == failed for diagnostic analysis.
	Error *string

	ReceivedAt  time.Time
	ProcessedAt *time.Time
}
