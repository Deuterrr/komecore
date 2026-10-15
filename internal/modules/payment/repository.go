package payment

import (
	"context"
	"time"

	"github.com/google/uuid"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/pagination"
)

var (
	PaymentMethodSortLatest pagination.SortKey = "latest"
	PaymentMethodSortName   pagination.SortKey = "name"
	PaymentMethodSortCode   pagination.SortKey = "code"
	PaymentMethodSortType   pagination.SortKey = "type"
)

type Repository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*Payment, error)

	GetByOrderID(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*Payment, error)

	GetByOrderIDForUpdate(
		ctx context.Context,
		exec transaction.Executor,
		orderID uuid.UUID,
	) (*Payment, error)

	ListByOrderIDs(
		ctx context.Context,
		exec transaction.Executor,
		orderIDs []uuid.UUID,
	) ([]Payment, error)

	UpdateStatus(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		status PaymentStatus,
	) error

	Save(
		ctx context.Context,
		exec transaction.Executor,
		payment Payment,
	) error

	ListPendingGateway(
		ctx context.Context,
		exec transaction.Executor,
		since time.Time,
	) ([]Payment, error)

	ListPastDuePending(
		ctx context.Context,
		exec transaction.Executor,
		now time.Time,
		limit int,
	) ([]Payment, error)
}

type MethodRepository interface {
	Save(
		ctx context.Context,
		exec transaction.Executor,
		method PaymentMethod,
	) error

	FindByName(
		ctx context.Context,
		exec transaction.Executor,
		name string,
	) (*PaymentMethod, error)

	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*PaymentMethod, error)

	ListAll(
		ctx context.Context,
		exec transaction.Executor,
		sorts pagination.Sorts,
	) ([]PaymentMethod, error)
}

type EventRepository interface {
	GetByID(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) (*PaymentEvent, error)

	ListByPaymentID(
		ctx context.Context,
		exec transaction.Executor,
		paymentID uuid.UUID,
	) ([]PaymentEvent, error)

	Create(
		ctx context.Context,
		exec transaction.Executor,
		event PaymentEvent,
	) error
}

type InstructionRepository interface {
	GetByPaymentMethodID(
		ctx context.Context,
		exec transaction.Executor,
		methodID uuid.UUID,
	) (*PaymentInstruction, error)

	Save(
		ctx context.Context,
		exec transaction.Executor,
		instruction PaymentInstruction,
	) error
}

type ChannelDataRepository interface {
	Save(
		ctx context.Context,
		exec transaction.Executor,
		data PaymentChannelData,
	) error

	GetByPaymentID(
		ctx context.Context,
		exec transaction.Executor,
		paymentID uuid.UUID,
	) (*PaymentChannelData, error)

	ListByPaymentIDs(
		ctx context.Context,
		exec transaction.Executor,
		paymentIDs []uuid.UUID,
	) (map[uuid.UUID]*PaymentChannelData, error)
}

type WebhookEventRepository interface {
	Upsert(
		ctx context.Context,
		exec transaction.Executor,
		event PaymentWebhookEvent,
	) (*PaymentWebhookEvent, error)

	MarkProcessed(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
	) error

	MarkFailed(
		ctx context.Context,
		exec transaction.Executor,
		id uuid.UUID,
		errMsg string,
	) error
}

type PaymentRepository = Repository
type PaymentMethodRepository = MethodRepository
type PaymentEventRepository = EventRepository
type PaymentInstructionRepository = InstructionRepository
type PaymentChannelDataRepository = ChannelDataRepository
type PaymentWebhookEventRepository = WebhookEventRepository
