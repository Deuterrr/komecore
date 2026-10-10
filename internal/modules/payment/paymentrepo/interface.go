package paymentrepo

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/pagination"

	"github.com/google/uuid"
)

type PaymentRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*paymentdomain.Payment, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*paymentdomain.Payment, error)

	// GetByOrderIDForUpdate retrieves the payment record and acquires a row-level exclusive lock
	// (SELECT ... FOR UPDATE) to prevent concurrent state transitions.
	//
	// MUST be called inside an active transaction.
	GetByOrderIDForUpdate(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*paymentdomain.Payment, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]paymentdomain.Payment, error)

	UpdateStatus(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status paymentdomain.PaymentStatus,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		payment paymentdomain.Payment,
	) error

	// ListPendingGateway returns gateway-provider payments
	// still in 'pending' status whose provider_order_id is set
	// and whose created_at is >= since.
	//
	// Used by the reconciliation job to find payments
	// missed by webhooks.
	ListPendingGateway(
		ctx context.Context,
		exec transaction.Executor,
		since time.Time,
	) ([]paymentdomain.Payment, error)

	// ListPastDuePending returns pending payments whose expires_at
	// is non-null and less than or equal to now.
	ListPastDuePending(
		ctx context.Context,
		exec transaction.Executor,
		now time.Time,
		limit int,
	) ([]paymentdomain.Payment, error)
}

var (
	PaymentMethodSortLatest pagination.SortKey = "latest"
	PaymentMethodSortName   pagination.SortKey = "name"
	PaymentMethodSortCode   pagination.SortKey = "code"
	PaymentMethodSortType   pagination.SortKey = "type"
)

type PaymentMethodRepository interface {
	Save(
		ctx context.Context,
		exec transaction.Executor,
		method paymentdomain.PaymentMethod,
	) error

	FindByName(
		ctx context.Context,
		exec transaction.Executor,
		name string,
	) (*paymentdomain.PaymentMethod, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*paymentdomain.PaymentMethod, error)

	ListAll(
		ctx context.Context,
		exec transaction.Executor,
		sorts pagination.Sorts,
	) ([]paymentdomain.PaymentMethod, error)
}

type PaymentEventRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*paymentdomain.PaymentEvent, error)

	ListByPaymentID(
		ctx context.Context,
		exec transaction.Executor,
		paymentID uuid.UUID,
	) ([]paymentdomain.PaymentEvent, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		event paymentdomain.PaymentEvent,
	) error
}

type PaymentInstructionRepository interface {
	GetByPaymentMethodID(
		ctx context.Context,
		exec transaction.Executor,
		methodID uuid.UUID,
	) (*paymentdomain.PaymentInstruction, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		instruction paymentdomain.PaymentInstruction,
	) error
}

// PaymentChannelDataRepository persists gateway-returned payment channel
// details (QR strings, VA numbers, deep links) so they can be retrieved
// after the initial checkout response is discarded.
type PaymentChannelDataRepository interface {
	Save(
		ctx context.Context,
		exec transaction.Executor,
		data paymentdomain.PaymentChannelData,
	) error

	GetByPaymentID(
		ctx context.Context,
		exec transaction.Executor,
		paymentID uuid.UUID,
	) (*paymentdomain.PaymentChannelData, error)

	// ListByPaymentIDs returns channel data records indexed by payment ID.
	ListByPaymentIDs(
		ctx context.Context,
		exec transaction.Executor,
		paymentIDs []uuid.UUID,
	) (map[uuid.UUID]*paymentdomain.PaymentChannelData, error)
}

// PaymentWebhookEventRepository persists inbound gateway webhook payloads
// and tracks their processing lifecycle for idempotency and auditability.
type PaymentWebhookEventRepository interface {
	// Upsert inserts a new event row.
	// On conflict (order_id, transaction_status) it leaves the existing row
	// untouched and returns it, so the caller can inspect its current status
	// before deciding whether to re-process.
	Upsert(
		ctx context.Context,
		exec transaction.Executor,
		event paymentdomain.PaymentWebhookEvent,
	) (*paymentdomain.PaymentWebhookEvent, error)

	// MarkProcessed sets status = 'processed' and stamps processed_at.
	MarkProcessed(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	// MarkFailed sets status = 'failed' and records the error string.
	// The event will be re-attempted on the next webhook delivery from
	// the gateway for the same (order_id, transaction_status).
	MarkFailed(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		errMsg string,
	) error
}
