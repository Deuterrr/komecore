package order

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	OrderSortLatest pagination.SortKey = "latest"
	OrderSortNumber pagination.SortKey = "number"
	OrderSortTotal  pagination.SortKey = "total"
	OrderSortStatus pagination.SortKey = "status"
	OrderSortModify pagination.SortKey = "modify"
)

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

type PricingService interface {
	Calculate(
		ctx context.Context,
		exec transaction.Executor,
		input PricingInput,
	) (*PricingResult, error)
}

type Repository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Order, error)

	GetByIDForUpdate(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Order, error)

	GetByNumber(
		ctx context.Context,
		exec transaction.Executor,
		number string,
	) (*Order, error)

	UpdateStatus(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status OrderStatus,
	) error

	UpdateStatusWithSLA(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status OrderStatus,
		confirmedAt *time.Time,
		expiresAt *time.Time,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		order Order,
	) error

	FindOrders(
		ctx context.Context,
		exec transaction.Executor,
		params FindOrderParams,
	) ([]Order, int, error)

	SetConfirmedAndExpiry(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		confirmedAt time.Time,
		expiresAt time.Time,
	) error

	FindExpiredUnfulfilledOrders(
		ctx context.Context,
		exec transaction.Executor,
		now time.Time,
		limit int,
	) ([]Order, error)
}

type ItemRepository interface {
	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]OrderItem, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]OrderItem, error)

	ListByShipmentID(
		ctx context.Context,
		exec transaction.Executor,
		shipmentID uuid.UUID,
	) ([]OrderItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []OrderItem,
	) error

	AssignShipment(
		ctx context.Context,
		exec transaction.Executor,
		shipmentID uuid.UUID,
		itemIDs []uuid.UUID,
	) error
}

type InvoiceRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Invoice, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*Invoice, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		invoice Invoice,
	) error
}

type InvoiceItemRepository interface {
	ListByInvoiceID(
		ctx context.Context,
		exec transaction.Executor,
		invoiceID uuid.UUID,
	) ([]InvoiceItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []InvoiceItem,
	) error
}

type OrderRepository = Repository
type OrderItemRepository = ItemRepository
