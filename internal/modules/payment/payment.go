package payment

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	// Awaiting customer transfer, QR scan, or upstream gateway settlement.
	PaymentStatusPending PaymentStatus = "pending"
	// Funds captured and verified via upstream gateway webhook or reconciliation sync.
	PaymentStatusPaid PaymentStatus = "paid"
	// Unrecoverable processing failure or upstream gateway rejection.
	PaymentStatusFailed PaymentStatus = "failed"
	// Customer payment window timed out without settlement.
	PaymentStatusExpired PaymentStatus = "expired"
	// Aborted prior to settlement (e.g. order cancelled by customer or staff).
	PaymentStatusCancelled PaymentStatus = "cancelled"
	// Captured funds successfully remitted back to customer account.
	PaymentStatusRefunded PaymentStatus = "refunded"
	// Refund operation initiated with upstream processor; awaiting bank confirmation.
	PaymentStatusRefundPending PaymentStatus = "refund_pending"
	// Upstream refund request rejected or failed execution.
	PaymentStatusRefundFailed PaymentStatus = "refund_failed"
)

// Upstream payment processor or multi-channel gateway aggregator discriminator.
type PaymentProvider string

const (
	// Aggregated multi-channel payment gateway (e.g. Midtrans, Xendit).
	PaymentProviderGateway PaymentProvider = "gateway"
)

// Aggregate root binding an order to external gateway charges, tracking settlement, and refund lifecycle.
//
// Invariant: Amount is stored as an integer in fractional currency / lowest denomination (e.g. IDR Rupiah or cents).
type Payment struct {
	ID uuid.UUID

	OrderID  uuid.UUID
	MethodID uuid.UUID

	Provider string

	// Gateway's internal transaction ID once acknowledged by upstream processor.
	ProviderPaymentID *string
	// Merchant-side correlation ID passed upstream; presence identifies an active gateway session.
	ProviderOrderID *string

	Amount int64

	Status PaymentStatus

	ExpiresAt *time.Time
	PaidAt    *time.Time

	CreatedAt time.Time
	UpdatedAt *time.Time
}
