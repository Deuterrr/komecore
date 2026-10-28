package payment

import (
	"time"

	"github.com/google/uuid"
)

// Persists transient gateway charge responses to guarantee idempotent recovery
// and customer retrieval across browser reloads.
type PaymentChannelData struct {
	ID        uuid.UUID
	PaymentID uuid.UUID

	// Rail classification.
	ChannelType PaymentMethodType
	// Merchant-facing brand label from gateway response (e.g. "BCA Virtual Account", "QRIS").
	DisplayName string

	AccountNumber *string
	QRString      *string

	// External checkout URL or e-wallet deep link for mobile app switching.
	RedirectURL *string
	// Actionable link retained for backward compatibility with legacy mobile SDKs.
	ActionURL *string
	// Provider-specific metadata (biller_code, acquirer, settlement window).
	Metadata map[string]any

	ExpiresAt *time.Time
	CreatedAt time.Time
}
