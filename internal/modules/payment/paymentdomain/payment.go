package paymentdomain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusPaid          PaymentStatus = "paid"
	PaymentStatusFailed        PaymentStatus = "failed"
	PaymentStatusExpired       PaymentStatus = "expired"
	PaymentStatusCancelled     PaymentStatus = "cancelled"
	PaymentStatusRefunded      PaymentStatus = "refunded"
	PaymentStatusRefundPending PaymentStatus = "refund_pending"
	PaymentStatusRefundFailed  PaymentStatus = "refund_failed"
)

type PaymentProvider string

const (
	PaymentProviderGateway PaymentProvider = "gateway"
)

type Payment struct {
	ID uuid.UUID

	OrderID  uuid.UUID
	MethodID uuid.UUID

	Provider string

	ProviderPaymentID *string
	ProviderOrderID   *string

	Amount int64

	Status PaymentStatus

	ExpiresAt *time.Time
	PaidAt    *time.Time

	CreatedAt time.Time
	UpdatedAt *time.Time
}
