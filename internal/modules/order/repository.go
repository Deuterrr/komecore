package order

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	// Orders records by descending creation timestamp (default display order).
	OrderSortLatest pagination.SortKey = "latest"
	// Orders records alphabetically by human-readable unique order code.
	OrderSortNumber pagination.SortKey = "number"
	// Orders records by monetary total amount.
	OrderSortTotal pagination.SortKey = "total"
	// Orders records by lifecycle status machine enum value.
	OrderSortStatus pagination.SortKey = "status"
	// Orders records by descending last modification timestamp.
	OrderSortModify pagination.SortKey = "modify"
)

// FindOrderParams encapsulates multi-tenant filtering, date ranges, and pagination for order queries.
type FindOrderParams struct {
	ID         *uuid.UUID
	Number     *string
	CustomerID *uuid.UUID
	ShopID     *uuid.UUID
	ShopIDs    []uuid.UUID
	Status     *string
	Statuses   []string
	FromDate   *time.Time
	ToDate     *time.Time

	Pagination pagination.Pagination
	Sorts      pagination.Sorts
}

// PricingService evaluates multi-shop basket pricing, courier logistics rates, and promotional discounts.
type PricingService interface {
	Calculate(ctx context.Context, exec transaction.Executor, input PricingInput) (*PricingResult, error)
}

// Repository defines persistence operations for order lifecycle aggregates, status transitions,
// and fulfillment SLA tracking across database transaction boundaries.
type Repository interface {
	// Resolves an order aggregate by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Order, error)
	// Acquires a pessimistic row lock (`FOR UPDATE`) to prevent race conditions
	// between concurrent webhook deliveries, manual cancellations, and status reconciliations.
	GetByIDForUpdate(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Order, error)
	// Looks up an order by human-readable unique order number.
	GetByNumber(ctx context.Context, exec transaction.Executor, number string) (*Order, error)

	// Persists an atomic state transition, validating against finite state machine lifecycle invariants.
	UpdateStatus(ctx context.Context, exec transaction.Executor, id uuid.UUID, status OrderStatus) error
	// Transitions status while stamping fulfillment SLA tracking timestamps (confirmed_at, expires_at).
	UpdateStatusWithSLA(ctx context.Context, exec transaction.Executor, id uuid.UUID, status OrderStatus, confirmedAt *time.Time, expiresAt *time.Time) error

	// Persists an order aggregate, inserting a new record or updating state within transaction boundaries.
	Save(ctx context.Context, exec transaction.Executor, order Order) error

	// Queries paginated orders matching search filters, returning items and total matching count.
	FindOrders(ctx context.Context, exec transaction.Executor, params FindOrderParams) ([]Order, int, error)

	// Sets payment confirmation timestamp and initializes 72h merchant fulfillment handling countdown.
	SetConfirmedAndExpiry(ctx context.Context, exec transaction.Executor, id uuid.UUID, confirmedAt time.Time, expiresAt time.Time) error

	// Fetches confirmed and processing orders past their fulfillment deadline for automated cancellation.
	FindExpiredUnfulfilledOrders(ctx context.Context, exec transaction.Executor, now time.Time, limit int) ([]Order, error)
}

// ItemRepository defines persistence operations for order line items, basket snapshot details,
// and courier consignment package assignments.
type ItemRepository interface {
	// Queries all purchased line items belonging to a specific order.
	ListByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]OrderItem, error)
	// Batch-loads line items across multiple orders to eliminate N+1 roundtrips.
	ListByOrderIDs(ctx context.Context, exec transaction.Executor, orderIDs []uuid.UUID) ([]OrderItem, error)
	// Queries line items fulfilled within a specific courier consignment shipment package.
	ListByShipmentID(ctx context.Context, exec transaction.Executor, shipmentID uuid.UUID) ([]OrderItem, error)

	// Batch-inserts order line items, preserving snapshot product name, variants, price, and shipping fee.
	SaveBulk(ctx context.Context, exec transaction.Executor, items []OrderItem) error

	// Associates line item records with a shipment consignment upon merchant parcel dispatch.
	AssignShipment(ctx context.Context, exec transaction.Executor, shipmentID uuid.UUID, itemIDs []uuid.UUID) error
}

// InvoiceRepository defines persistence operations for legal tax invoices, sequential billing records,
// and commercial purchase receipts.
type InvoiceRepository interface {
	// Resolves an invoice record by primary identifier within transactional context.
	GetByID(ctx context.Context, exec transaction.Executor, id uuid.UUID) (*Invoice, error)
	// Looks up the commercial tax invoice issued for a specific order.
	GetByOrderID(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Invoice, error)

	// Persists an invoice record with customer billing details, tax calculations, and payment references.
	Save(ctx context.Context, exec transaction.Executor, invoice Invoice) error
}

// InvoiceItemRepository defines persistence operations for itemized billing line items recorded on commercial invoices.
type InvoiceItemRepository interface {
	// Queries all itemized billing rows linked to a commercial tax invoice.
	ListByInvoiceID(ctx context.Context, exec transaction.Executor, invoiceID uuid.UUID) ([]InvoiceItem, error)

	// Batch-inserts itemized billing line items associated with an invoice.
	SaveBulk(ctx context.Context, exec transaction.Executor, items []InvoiceItem) error
}
