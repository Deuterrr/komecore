package payment

import (
	"time"

	"github.com/google/uuid"
)

// Markdown-formatted checkout guidance rendered dynamically across customer sessions.
type PaymentInstruction struct {
	ID              uuid.UUID
	PaymentMethodID uuid.UUID

	Content string

	CreatedAt time.Time
}
