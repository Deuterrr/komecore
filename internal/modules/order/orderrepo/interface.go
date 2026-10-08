package orderrepo

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"

	"github.com/google/uuid"
)

type PricingService interface {
	Calculate(
		ctx context.Context,
		exec transaction.Executor,
		input PricingInput,
	) (*PricingResult, error)
}

type OrderRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*orderdomain.Order, error)

	GetByNumber(
		ctx context.Context,
		exec transaction.Executor,
		number string,
	) (*orderdomain.Order, error)

	UpdateStatus(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status orderdomain.OrderStatus,
	) error

	UpdateStatusWithSLA(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status orderdomain.OrderStatus,
		confirmedAt *time.Time,
		expiresAt *time.Time,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		order orderdomain.Order,
	) error

	FindOrders(
		ctx context.Context,
		exec transaction.Executor,
		params FindOrderParams,
	) ([]orderdomain.Order, int, error)

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
	) ([]orderdomain.Order, error)
}

type OrderItemRepository interface {
	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]orderdomain.OrderItem, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]orderdomain.OrderItem, error)

	ListByShipmentID(
		ctx context.Context,
		exec transaction.Executor,
		shipmentID uuid.UUID,
	) ([]orderdomain.OrderItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []orderdomain.OrderItem,
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
	) (*orderdomain.Invoice, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*orderdomain.Invoice, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		invoice orderdomain.Invoice,
	) error
}

type InvoiceItemRepository interface {
	ListByInvoiceID(
		ctx context.Context,
		exec transaction.Executor,
		invoiceID uuid.UUID,
	) ([]orderdomain.InvoiceItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []orderdomain.InvoiceItem,
	) error
}
