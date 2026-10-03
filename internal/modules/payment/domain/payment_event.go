package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentEventStatus string

const (
	PaymentEventStatusPending    PaymentEventStatus = "pending"
	PaymentEventStatusSettlement PaymentEventStatus = "settlement"
	PaymentEventStatusRefund     PaymentEventStatus = "refund"
	PaymentEventStatusExpire     PaymentEventStatus = "expire"
)

type PaymentEvent struct {
	ID uuid.UUID

	PaymentID uuid.UUID

	EventName string

	Payload json.RawMessage

	CreatedAt time.Time
}
