package paymentdomain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentChannelData holds the gateway-returned payment channel details
// (QR string, VA number, deep-link URL) persisted after a charge so they
// can be retrieved on any subsequent request — e.g. after a page refresh.
type PaymentChannelData struct {
	ID uuid.UUID

	PaymentID uuid.UUID

	// ChannelType is the payment method type, using the domain constants:
	// TypeBankTransfer | TypeEWallet | TypeQRCode
	ChannelType PaymentMethodType

	// DisplayName is the human-readable label returned by the gateway,
	// e.g. "QRIS", "GoPay", "BCA Virtual Account".
	DisplayName string

	// AccountNumber stores virtual account numbers or Mandiri Bill Keys.
	AccountNumber *string

	// QRString stores raw EMVCo QR code payloads for dynamic QRIS rendering.
	QRString *string

	// RedirectURL stores external checkout links or e-wallet deep links.
	RedirectURL *string

	// ActionURL is the legacy actionable value (retained for backward compatibility).
	ActionURL *string

	// Metadata holds arbitrary provider-specific context (biller_code, acquirer, etc.).
	Metadata map[string]any

	ExpiresAt *time.Time

	CreatedAt time.Time
}
