package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	inventoryRepo "komecore/internal/modules/inventory/repository"
	paymentDomain "komecore/internal/modules/payment/domain"
	paymentRepo "komecore/internal/modules/payment/repository"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"
)

// SyncPendingPaymentsUsecase is the background reconciliation
// job usecase.
//
// It scans for gateway payments that are still 'pending'
// within a look-back window and queries Midtrans directly
// for their current status.
//
// If the status has changed, it drives the payment through
// the existing webhook processing usecase — including the
// idempotency gate — so no double-processing can
// occur even if a late webhook also arrives.
type SyncPendingPaymentsUsecase struct {
	paymentRepo    paymentRepo.PaymentRepository
	paymentGateway paymentgateway.Provider
	processWebhook *ProcessPaymentWebhookUsecase
	executor       transaction.Executor
	logger         applogger.Logger
	lookbackWindow time.Duration
	transactor     transaction.Transactor
	orderMgr       OrderPaymentManager
	inventoryRepo  inventoryRepo.InventoryRepository
}

func NewSyncPendingPaymentsUsecase(
	paymentRepo paymentRepo.PaymentRepository,
	paymentGateway paymentgateway.Provider,
	processWebhook *ProcessPaymentWebhookUsecase,
	executor transaction.Executor,
	logger applogger.Logger,
	lookbackWindow time.Duration,
	transactor transaction.Transactor,
	orderMgr OrderPaymentManager,
	inventoryRepo inventoryRepo.InventoryRepository,
) *SyncPendingPaymentsUsecase {
	return &SyncPendingPaymentsUsecase{
		paymentRepo:    paymentRepo,
		paymentGateway: paymentGateway,
		processWebhook: processWebhook,
		executor:       executor,
		logger:         logger,
		lookbackWindow: lookbackWindow,
		transactor:     transactor,
		orderMgr:       orderMgr,
		inventoryRepo:  inventoryRepo,
	}
}

// Execute reconciles pending payments with the upstream gateway within the
// configured lookback window.
//
// Processing is isolated per payment: a failure to reconcile one payment is
// logged and skipped without affecting other pending payments.
//
// Payments that have expired locally are reconciled through a best-effort
// gateway cancellation before their reserved inventory is released and their
// local payment state is finalized as expired.
func (u *SyncPendingPaymentsUsecase) Execute(ctx context.Context) {
	since := appclock.Now().Add(-u.lookbackWindow)
	var msg string

	payments, err := u.paymentRepo.ListPendingGateway(ctx, u.executor, since)
	if err != nil {
		msg = "failed to list pending gateway payments"
		u.logger.Error(ctx, msg,
			applogger.Field{Key: "error", Value: err.Error()},
		)
		return
	}
	if len(payments) == 0 {
		msg = "no pending gateway payments in window"
		u.logger.Info(ctx, msg)
		return
	}

	msg = "payment sync: starting reconciliation cycle"
	u.logger.Info(ctx, msg,
		applogger.Field{Key: "count", Value: len(payments)},
		applogger.Field{Key: "since", Value: since.Format(time.RFC3339)},
	)

	resolved := 0
	for _, payment := range payments {
		if payment.ProviderOrderID == nil {
			u.logger.Error(ctx, "gateway payment is missing provider order id",
				applogger.Field{Key: "payment_id", Value: payment.ID.String()},
			)
			continue
		}

		gatewayOrderID := *payment.ProviderOrderID
		result, err := u.paymentGateway.GetTransactionStatus(ctx, gatewayOrderID)
		if err != nil {
			msg = "failed to fetch transaction status"
			u.logger.Error(ctx, msg,
				applogger.Field{Key: "payment_id", Value: payment.ID.String()},
				applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
				applogger.Field{Key: "error", Value: err.Error()},
			)
			continue
		}

		// Skip if Midtrans still reports pending — nothing to do yet.
		if result.Status == paymentgateway.NotificationStatusPending {
			if payment.ExpiresAt != nil && appclock.Now().After(*payment.ExpiresAt) {
				msg = "payment has expired locally, cancelling at gateway and expiring locally"
				u.logger.Info(ctx, msg,
					applogger.Field{Key: "payment_id", Value: payment.ID.String()},
					applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
				)

				// Best effort cancel at gateway
				if err := u.paymentGateway.CancelTransaction(ctx, gatewayOrderID); err != nil {
					u.logger.Warn(ctx, "failed to cancel transaction at gateway, proceeding with local expiry",
						applogger.Field{Key: "payment_id", Value: payment.ID.String()},
						applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
						applogger.Field{Key: "error", Value: err.Error()},
					)
				}

				if err := u.expirePaymentLocally(ctx, payment); err != nil {
					u.logger.Error(ctx, "failed to locally expire payment",
						applogger.Field{Key: "payment_id", Value: payment.ID.String()},
						applogger.Field{Key: "error", Value: err.Error()},
					)
				} else {
					resolved++
				}
			}
			continue
		}

		// Drive resolved status directly through the webhook processing pipeline
		// without redundant outbound network calls.
		if err := u.processWebhook.Execute(ctx, ProcessPaymentWebhookInput{
			NotificationResult: result,
		}); err != nil {
			msg = "failed to process reconciled payment"
			u.logger.Error(ctx, msg,
				applogger.Field{Key: "payment_id", Value: payment.ID.String()},
				applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
				applogger.Field{Key: "gateway_status", Value: string(result.Status)},
				applogger.Field{Key: "error", Value: err.Error()},
			)
			continue
		}

		msg = "successfully reconciled payment"
		u.logger.Info(ctx, msg,
			applogger.Field{Key: "payment_id", Value: payment.ID.String()},
			applogger.Field{Key: "gateway_order_id", Value: gatewayOrderID},
			applogger.Field{Key: "gateway_status", Value: string(result.Status)},
		)
		resolved++
	}

	msg = "reconciliation cycle complete"
	u.logger.Info(ctx, msg,
		applogger.Field{Key: "total", Value: len(payments)},
		applogger.Field{Key: "resolved", Value: resolved},
		applogger.Field{Key: "skipped", Value: fmt.Sprintf("%d", len(payments)-resolved)},
	)
}

func (u *SyncPendingPaymentsUsecase) expirePaymentLocally(
	ctx context.Context,
	payment paymentDomain.Payment,
) error {
	return u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		if err := u.paymentRepo.UpdateStatus(ctx, exec,
			payment.ID,
			paymentDomain.PaymentStatusExpired,
		); err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		orderItems, err := u.orderMgr.ExpireOrderPayment(ctx, exec, payment.OrderID)
		if err != nil {
			return fmt.Errorf("failed to expire order payment: %w", err)
		}

		for _, item := range orderItems {
			if err := u.inventoryRepo.Release(ctx, exec,
				item.ProductID,
				item.ShopID,
				item.Quantity,
			); err != nil {
				if errors.Is(err, inventoryDomain.ErrInsufficientReserved) ||
					errors.Is(err, apperrors.ErrNotFound) {

					msg := "inventory anomaly during payment reconciliation expiry: reserved stock insufficient or missing"
					u.logger.Warn(ctx, msg,
						applogger.Field{Key: "payment_id", Value: payment.ID.String()},
						applogger.Field{Key: "order_id", Value: payment.OrderID.String()},
						applogger.Field{Key: "product_id", Value: item.ProductID.String()},
						applogger.Field{Key: "shop_id", Value: item.ShopID.String()},
						applogger.Field{Key: "requested_qty", Value: item.Quantity},
						applogger.Field{Key: "reason", Value: err.Error()},
					)
					continue
				}
				return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
			}
		}

		return nil
	})
}
