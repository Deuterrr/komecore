package payment

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Audit classification for gateway notifications and internal transaction lifecycle changes.
type PaymentEventStatus string

const (
	// Transaction initiated; awaiting customer action.
	PaymentEventStatusPending PaymentEventStatus = "pending"
	// Verified funds capture and transaction settlement.
	PaymentEventStatusSettlement PaymentEventStatus = "settlement"
	// Transaction reversal and funds remittance.
	PaymentEventStatusRefund PaymentEventStatus = "refund"
	// Customer payment deadline lapsed without settlement.
	PaymentEventStatusExpire PaymentEventStatus = "expire"
)

// Immutable audit log entry capturing raw gateway payloads and domain transitions.
//
// Invariant: Append-only ledger; entries cannot be altered or removed after creation.
type PaymentEvent struct {
	ID        uuid.UUID
	PaymentID uuid.UUID

	EventName string
	Payload   json.RawMessage

	CreatedAt time.Time
}
