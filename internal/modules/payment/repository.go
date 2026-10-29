package payment

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	// Orders channels by descending creation timestamp (default display order).
	PaymentMethodSortLatest pagination.SortKey = "latest"
	// Orders channels alphabetically by human-readable display name.
	PaymentMethodSortName pagination.SortKey = "name"
	// Orders channels alphabetically by internal channel code (e.g. bca_va, qris).
	PaymentMethodSortCode pagination.SortKey = "code"
	// Orders channels grouped by integration rail (bank_transfer, ewallet, qr_code).
	PaymentMethodSortType pagination.SortKey = "type"
)

// Persistence contract for financial transactions, settlement state, and reconciliation.
type Repository interface {
	// Resolves a transaction.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Payment, error)
	GetByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Payment, error)
	// Acquires a pessimistic row lock (`FOR UPDATE`) to prevent race conditions
	// between concurrent webhook deliveries, manual staff cancellations, and refund jobs.
	GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Payment, error)

	// Batch-loads transactions across multiple orders to eliminate N+1 database roundtrips.
	ListByOrderIDs(ctx context.Context, exec transaction.Executor, orderIDs []uuid.UUID) ([]Payment, error)

	// Persists a state transition atomically, stamping updated_at.
	UpdateStatus(ctx context.Context, exec transaction.Executor, id uuid.UUID, status PaymentStatus) error
	// Inserts a new record or replaces an existing row within the active database transaction.
	Save(ctx context.Context, exec transaction.Executor, payment Payment) error

	// Queries unsettled transactions created since the lookback window to reconcile with gateway status.
	//
	// Invariant: Presence of provider_order_id identifies an active upstream charge regardless of provider name.
	ListPendingGateway(ctx context.Context, exec transaction.Executor, since time.Time) ([]Payment, error)
	// Fetches pending records past their expiration deadline for cancellation and inventory hold release.
	ListPastDuePending(ctx context.Context, exec transaction.Executor, now time.Time, limit int) ([]Payment, error)
}

// Persistence contract for checkout options, pricing fee tiers, and soft-delete revival.
type MethodRepository interface {
	// Persists or updates a payment method.
	//
	// Invariant: Conflict on (code, provider, type) revives soft-deleted rows and preserves the canonical ID
	// to protect relational foreign key integrity on payment_instructions and payments.
	Save(ctx context.Context, exec transaction.Executor, method PaymentMethod) error

	// Matches an active, non-deleted payment method by case-insensitive name prefix.
	FindByName(ctx context.Context, exec transaction.Executor, name string) (*PaymentMethod, error)

	// Fetches a method by ID along with its linked instruction markdown if not soft-deleted.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*PaymentMethod, error)

	// Queries all active channels matching user pagination and sorting criteria.
	ListAll(ctx context.Context, exec transaction.Executor, sorts pagination.Sorts) ([]PaymentMethod, error)
}

// Append-only ledger for financial auditing and gateway event traceability.
type EventRepository interface {
	// Fetches a single audit entry by primary key.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*PaymentEvent, error)

	// Queries chronological history of all state changes and webhooks recorded for a transaction.
	ListByPaymentID(ctx context.Context, exec transaction.Executor, paymentID uuid.UUID) ([]PaymentEvent, error)

	// Appends an immutable audit event; rows cannot be updated or deleted after creation.
	Create(ctx context.Context, exec transaction.Executor, event PaymentEvent) error
}

// Persistence contract for customer checkout instructions and transfer steps.
type InstructionRepository interface {
	// Resolves the markdown guidance attached to a specific payment method.
	GetByPaymentMethodID(ctx context.Context, exec transaction.Executor, methodID uuid.UUID) (*PaymentInstruction, error)

	// Upserts the markdown checkout guidance for a payment method.
	Save(ctx context.Context, exec transaction.Executor, instruction PaymentInstruction) error
}

// Storage for transient gateway charge payloads (QRIS strings, VA numbers, app deep links).
type ChannelDataRepository interface {
	// Saves transient channel response data so customers can resume payment after a page refresh.
	Save(ctx context.Context, exec transaction.Executor, data PaymentChannelData) error

	// Resolves persisted checkout details (e.g. virtual account number or QR code) for a transaction.
	GetByPaymentID(ctx context.Context, exec transaction.Executor, paymentID uuid.UUID) (*PaymentChannelData, error)

	// Batch-loads channel details across multiple payments to prevent N+1 query overhead.
	ListByPaymentIDs(ctx context.Context, exec transaction.Executor, paymentIDs []uuid.UUID) (map[uuid.UUID]*PaymentChannelData, error)
}

// Idempotent ingestion pipeline for incoming asynchronous gateway webhook notifications.
type WebhookEventRepository interface {
	// Inserts a new webhook delivery row.
	// On conflict (gateway_order_id, transaction_status) the existing row is returned
	// unchanged, allowing the caller to inspect its status before deciding whether to re-process.
	Upsert(ctx context.Context, exec transaction.Executor, event PaymentWebhookEvent) (*PaymentWebhookEvent, error)

	// Marks the event as successfully applied to domain aggregates, recording processed_at timestamp.
	MarkProcessed(ctx context.Context, exec transaction.Executor, id uuid.UUID) error
	// Records processing failure and error message to unblock retries on redelivery.
	MarkFailed(ctx context.Context, exec transaction.Executor, id uuid.UUID, errMsg string) error
}
