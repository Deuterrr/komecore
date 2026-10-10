package paymentgateway

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ChargeItem struct {
	ID       string
	Name     string
	Quantity int
	Price    int64
}

type ChargeRequest struct {
	// PaymentID is the internal Payment UUID
	// — used as the idempotency key external
	// order-reference sent to the gateway
	PaymentID uuid.UUID

	// OrderID is the staff's order reference
	// exposed to the gateway
	OrderID uuid.UUID

	// Amount is the gross amount
	// in the smallest currency unit
	// (e.g. IDR cents)
	Amount int64

	// PaymentType is the gateway-specific
	// payment channel identifier
	// (e.g. "bank_transfer", "gopay", "qris").
	PaymentType string

	// BankCode is only required when
	// PaymentType is "bank_transfer"
	BankCode string

	// CustomerName is optional;
	// included in payment instructions
	// when available.
	CustomerName string

	// optional
	CustomerEmail string
	CustomerPhone string

	// ExpiresAt is the desired
	// expiry time for the payment
	//
	// When zero the gateway will apply
	// its own default
	ExpiresAt time.Time

	// Items lists the individual products, shipping fees, or
	// adjustments that comprise the total transaction amount.
	Items []ChargeItem
}

// PaymentInstruction is a single actionable
// piece of payment guidance returned by the gateway
// (e.g. a virtual-account number or a QR string)
type PaymentInstruction struct {
	Type  string // "bank_transfer" | "qris" | "ewallet" | …
	Label string // human-readable label, e.g. "BCA Virtual Account"
	Value string // the account number, QR string, or deep-link URL
}

type ChargeResponse struct {
	// GatewayTransactionID is the unique
	// transaction identifier assigned by the
	// payment gateway
	// (e.g. Midtrans transaction_id)
	GatewayTransactionID string

	// GatewayOrderID is the order identifier
	// echoed back by the gateway
	GatewayOrderID string

	// PaymentType is the resolved payment-channel type
	// (as returned by the gateway,
	// may differ from the requested type for aliases)
	PaymentType string

	// GrossAmount is the total amount
	// as confirmed by the gateway
	GrossAmount int64

	// Status is the initial gateway
	// status of the transaction
	Status string

	// Instructions carries human-readable
	// payment instructions
	// (VA numbers, QR strings, deep-link URLs, etc.)
	Instructions []PaymentInstruction

	// AccountNumber carries the bank virtual account number or bill key, if applicable.
	AccountNumber *string

	// QRString carries the raw QRIS or payment QR payload, if applicable.
	QRString *string

	// RedirectURL carries the e-wallet or checkout deep-link URL, if applicable.
	RedirectURL *string

	// ExpiresAt is the transaction expiry time
	// reported by the gateway
	//
	// May be zero when the gateway does not
	// return one
	ExpiresAt time.Time
}

type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "pending"
	NotificationStatusSettlement NotificationStatus = "settlement"
	NotificationStatusExpire     NotificationStatus = "expire"
	NotificationStatusCancel     NotificationStatus = "cancel"
	NotificationStatusDeny       NotificationStatus = "deny"
	NotificationStatusRefund     NotificationStatus = "refund"
	NotificationStatusChallenge  NotificationStatus = "challenge"
)

// NotificationResult is the normalised result
// of parsing a gateway webhook
type NotificationResult struct {
	// GatewayTransactionID is the gateway-side
	// transaction identifier
	GatewayTransactionID string

	// GatewayOrderID maps back to the internal PaymentID
	// sent as the order reference at charge time
	GatewayOrderID string

	// Status is the normalised transaction status
	Status NotificationStatus

	// GrossAmount is the amount confirmed
	// by the gateway
	GrossAmount int64

	// FraudStatus is the anti-fraud result
	// Empty when not applicable
	// e.g. ("accept", "challenge", "deny")
	FraudStatus string

	// RawStatus is the unmodified status string
	// from the gateway
	RawStatus string
}

// NotificationPayload is the raw webhook
// body forwarded from the gateway
//
// It is intentionally generic
// so each provider can deserialise it itself
type NotificationPayload map[string]any

type AllowedPaymentMethod struct {
	Code          string
	Name          string
	Type          string // "bank_transfer" | "ewallet" | "qr_code"
	FeeType       string // "flat" | "percentage" | "mixed"
	FeeFixed      int64
	FeePercentage float64
	FeeMax        *int64
	Description   string
}

type RefundRequest struct {
	GatewayOrderID string
	RefundAmount   int64
	Reason         string
}

type RefundResponse struct {
	GatewayTransactionID string
	GatewayOrderID       string
	RefundAmount         int64
	Status               string
}

// ErrInvalidSignature indicates that an inbound webhook failed cryptographic signature verification.
var ErrInvalidSignature = errors.New("paymentgateway: invalid webhook signature")

// Provider defines the vendor-agnostic contract for communicating with an external
// payment gateway (e.g. Midtrans, Xendit).
//
// All implementations MUST be thread-safe, handle context cancellation, and return
// domain-friendly errors without leaking third-party SDK types across the boundary.
type Provider interface {
	// Name returns the unique system identifier for this provider (e.g. "midtrans").
	Name() string

	// AllowedPaymentMethods returns the static catalogue of payment channels supported
	// by this provider along with their baseline fee structures.
	AllowedPaymentMethods() []AllowedPaymentMethod

	// Supports reports whether this provider is configured to handle the given payment channel code
	// (e.g. "bca_va", "gopay", "qris").
	Supports(code string) bool

	// Charge initiates a new charge transaction with the upstream payment gateway.
	//
	// Preconditions:
	//   - req.PaymentID must be unique; it acts as the gateway idempotency key (order_id).
	//   - req.Amount must be positive and match the sum of req.Items plus any fee adjustment.
	//
	// Guarantees:
	//   - Safe to retry if the gateway supports idempotent order references.
	//   - Returns an error if the channel is unsupported or the gateway rejects the payload.
	Charge(
		ctx context.Context,
		req ChargeRequest,
	) (*ChargeResponse, error)

	// ParseNotification verifies and parses an inbound webhook notification payload.
	//
	// Security & Performance Contract:
	//   - MUST verify the cryptographic signature (e.g. SHA-512) before processing.
	//   - MUST NOT execute redundant outbound HTTP status requests if the signature is valid.
	//   - Returns ErrInvalidSignature if the payload has been tampered with.
	ParseNotification(
		ctx context.Context,
		payload NotificationPayload,
	) (*NotificationResult, error)

	// GetTransactionStatus queries the gateway directly for the definitive status of an order.
	//
	// Used exclusively by the background reconciliation worker (SyncPendingPayments) and the
	// customer-initiated manual sync (CheckPaymentStatus) to resolve stuck pending states.
	GetTransactionStatus(
		ctx context.Context,
		gatewayOrderID string,
	) (*NotificationResult, error)

	// CancelTransaction instructs the gateway to cancel or void an active pending payment.
	//
	// Safe to call if the transaction is already expired or canceled; implementations should
	// treat 412/404 from upstream as a non-fatal outcome.
	CancelTransaction(
		ctx context.Context,
		gatewayOrderID string,
	) error

	// RefundTransaction issues a partial or full monetary refund for a settled transaction.
	//
	// Preconditions:
	//   - The referenced transaction must be in NotificationStatusSettlement.
	//   - req.RefundAmount must not exceed the original charged amount.
	RefundTransaction(
		ctx context.Context,
		req RefundRequest,
	) (*RefundResponse, error)
}
