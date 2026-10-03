package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentInstruction is the payment guidance record
// associated with a payment (1-to-1)
//
// Content is a Markdown-formatted string containing
// all human-readable steps the customer must follow
// to complete the payment — either composed from gateway
// charge instructions or from a manual payment account
type PaymentInstruction struct {
	ID              uuid.UUID
	PaymentMethodID uuid.UUID

	Content string

	CreatedAt time.Time
}
