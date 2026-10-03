package repository

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/domain"

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
	) (*domain.Order, error)

	GetByNumber(
		ctx context.Context,
		exec transaction.Executor,
		number string,
	) (*domain.Order, error)

	UpdateStatus(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status domain.OrderStatus,
	) error

	UpdateStatusWithSLA(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status domain.OrderStatus,
		confirmedAt *time.Time,
		expiresAt *time.Time,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		order domain.Order,
	) error

	FindOrders(
		ctx context.Context,
		exec transaction.Executor,
		params FindOrderParams,
	) ([]domain.Order, int, error)

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
	) ([]domain.Order, error)
}

type OrderItemRepository interface {
	ListByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) ([]domain.OrderItem, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]domain.OrderItem, error)

	ListByShipmentID(
		ctx context.Context,
		exec transaction.Executor,
		shipmentID uuid.UUID,
	) ([]domain.OrderItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []domain.OrderItem,
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
	) (*domain.Invoice, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*domain.Invoice, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		invoice domain.Invoice,
	) error
}

type InvoiceItemRepository interface {
	ListByInvoiceID(
		ctx context.Context,
		exec transaction.Executor,
		invoiceID uuid.UUID,
	) ([]domain.InvoiceItem, error)

	SaveBulk(
		ctx context.Context,
		exec transaction.Executor,
		items []domain.InvoiceItem,
	) error
}
