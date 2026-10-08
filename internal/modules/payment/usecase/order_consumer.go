package usecase

import (
	"context"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
)

type OrderInfo struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Number     string
	Total      int64
}

type OrderItemInfo struct {
	ProductID uuid.UUID
	ShopID    uuid.UUID
	Quantity  int
}

type OrderPaymentManager interface {
	GetOrderForPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*OrderInfo, error)
	GetInvoiceNumber(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (string, error)
	ConfirmOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID, confirmedAt time.Time, handlingWindow time.Duration) ([]OrderItemInfo, error)
	ExpireOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error)
	CancelOrderPayment(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) ([]OrderItemInfo, error)
}
