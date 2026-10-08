package paymentusecase

import (
	"context"
	"fmt"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"

	"github.com/google/uuid"
)

// CheckPaymentStatusUsecase is the customer-
// triggered payment sync.
//
// When a customer's order appears stuck as 'pending'
// after they have paid, they can call this usecase to immediately
// query Midtrans for the current status and resolve the payment â€”
// without waiting for the next background reconciliation tick.
type CheckPaymentStatusUsecase struct {
	orderMgr       OrderPaymentManager
	repository     paymentrepo.PaymentRepository
	paymentGateway paymentgateway.Provider
	processWebhook *ProcessPaymentWebhookUsecase
	executor       transaction.Executor
}

func NewCheckPaymentStatusUsecase(
	orderMgr OrderPaymentManager,
	repository paymentrepo.PaymentRepository,
	paymentGateway paymentgateway.Provider,
	processWebhook *ProcessPaymentWebhookUsecase,
	executor transaction.Executor,
) *CheckPaymentStatusUsecase {
	return &CheckPaymentStatusUsecase{
		orderMgr:       orderMgr,
		repository:     repository,
		paymentGateway: paymentGateway,
		processWebhook: processWebhook,
		executor:       executor,
	}
}

type CheckPaymentStatusInput struct {
	OrderID    uuid.UUID
	CustomerID uuid.UUID
}

type CheckPaymentStatusResult struct {
	// Status is the payment status after the check
	// (may be unchanged if Midtrans still reports pending).
	Status paymentdomain.PaymentStatus

	// Synced is true when the status was resolved
	// from Midtrans during this call
	// (i.e. it was pending before and is now terminal).
	Synced bool
}

// Execute performs a synchronous, customer-triggered payment verification
// against the external gateway to resolve payment states that may temporarily
// lag behind asynchronous webhook notifications.
//
// The operation is idempotent for payments already in a terminal state and
// avoids unnecessary gateway I/O in those cases.
//
// For unresolved payments, the gateway's current status is reconciled through
// the existing payment processing pipeline, ensuring that resolved outcomes
// consistently update payment, inventory, and order state.
func (u *CheckPaymentStatusUsecase) Execute(
	ctx context.Context,
	input CheckPaymentStatusInput,
) (*CheckPaymentStatusResult, error) {
	order, err := u.orderMgr.GetOrderForPayment(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: retrieve order: %w", err)
	}
	if order == nil {
		return nil, apperrors.NewNotFound("order not found")
	}
	if order.CustomerID != input.CustomerID {
		return nil, apperrors.NewNotFound("order not found")
	}

	payment, err := u.repository.GetByOrderID(ctx, u.executor, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: retrieve payment: %w", err)
	}
	if payment == nil {
		return nil, apperrors.NewNotFound("payment not found for order")
	}
	// Only gateway payments with a ProviderOrderID can be synced.
	if payment.ProviderOrderID == nil {
		return &CheckPaymentStatusResult{Status: payment.Status, Synced: false}, nil
	}
	// Only pending payments need a sync â€” return early for any terminal state.
	if payment.Status != paymentdomain.PaymentStatusPending {
		return &CheckPaymentStatusResult{Status: payment.Status, Synced: false}, nil
	}

	result, err := u.paymentGateway.GetTransactionStatus(ctx, *payment.ProviderOrderID)
	if err != nil {
		return nil, fmt.Errorf("check payment status: gateway status check failed: %w", err)
	}

	if result.Status == paymentgateway.NotificationStatusPending {
		return &CheckPaymentStatusResult{
			Status: payment.Status,
			Synced: false,
		}, nil
	}

	// Status has resolved â€” drive it through the standard
	// webhook processing pipeline directly without redundant outbound calls.
	if err := u.processWebhook.Execute(ctx, ProcessPaymentWebhookInput{
		NotificationResult: result,
	}); err != nil {
		return nil, fmt.Errorf("check payment status: process resolved status: %w", err)
	}

	resolvedStatus := mapGatewayStatus(result.Status)
	res := CheckPaymentStatusResult{
		Status: resolvedStatus,
		Synced: true,
	}

	return &res, nil
}

// mapGatewayStatus converts a gateway NotificationStatus to the domain
// PaymentStatus that the processWebhook usecase would have set.
func mapGatewayStatus(s paymentgateway.NotificationStatus) paymentdomain.PaymentStatus {
	switch s {
	case paymentgateway.NotificationStatusSettlement:
		return paymentdomain.PaymentStatusPaid
	case paymentgateway.NotificationStatusExpire:
		return paymentdomain.PaymentStatusExpired
	case paymentgateway.NotificationStatusCancel, paymentgateway.NotificationStatusDeny:
		return paymentdomain.PaymentStatusCancelled
	default:
		return paymentdomain.PaymentStatusPending
	}
}
