package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookEventStatus tracks the processing lifecycle of an inbound webhook.
type WebhookEventStatus string

const (
	// WebhookEventStatusReceived means the payload has been persisted
	// but not yet processed.
	WebhookEventStatusReceived WebhookEventStatus = "received"

	// WebhookEventStatusProcessed means the webhook was applied to
	// the payment domain successfully.
	WebhookEventStatusProcessed WebhookEventStatus = "processed"

	// WebhookEventStatusFailed means processing failed.
	// The event will be re-attempted on the next webhook delivery
	// from the gateway for the same (order_id, transaction_status).
	WebhookEventStatusFailed WebhookEventStatus = "failed"
)

// PaymentWebhookEvent is a persisted record of a single inbound
// gateway webhook notification.
//
// The pair (OrderID, TransactionStatus) acts as the idempotency key,
// enforced by a UNIQUE constraint in the database.
// Each unique status transition from the gateway is stored and
// processed exactly once.
type PaymentWebhookEvent struct {
	ID uuid.UUID

	// OrderID is the gateway-side order identifier extracted from
	// the raw webhook payload (not the internal UUID).
	OrderID string

	// TransactionStatus is the raw transaction_status field from
	// the gateway payload, used together with OrderID as the
	// idempotency key.
	TransactionStatus string

	// Payload is the full raw webhook body as received from the gateway.
	Payload json.RawMessage

	Status WebhookEventStatus

	// Error is populated when Status == WebhookEventStatusFailed.
	Error *string

	ReceivedAt  time.Time
	ProcessedAt *time.Time
}
