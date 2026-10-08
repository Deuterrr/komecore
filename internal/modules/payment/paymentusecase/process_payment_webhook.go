package paymentusecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory/inventorydomain"
	"komecore/internal/modules/inventory/inventoryrepo"
	"komecore/internal/modules/payment/paymentdomain"
	"komecore/internal/modules/payment/paymentrepo"
	appclock "komecore/pkg/clock"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type ProcessPaymentWebhookUsecase struct {
	repository       paymentrepo.PaymentRepository
	paymentEventRepo paymentrepo.PaymentEventRepository
	webhookEventRepo paymentrepo.PaymentWebhookEventRepository
	orderMgr         OrderPaymentManager
	inventoryRepo    inventoryrepo.InventoryRepository
	paymentGateway   paymentgateway.Provider
	auditLogger      applogger.AuditLogger
	transactor       transaction.Transactor
	executor         transaction.Executor
}

func NewProcessPaymentWebhookUsecase(
	repository paymentrepo.PaymentRepository,
	paymentEventRepo paymentrepo.PaymentEventRepository,
	webhookEventRepo paymentrepo.PaymentWebhookEventRepository,
	orderMgr OrderPaymentManager,
	inventoryRepo inventoryrepo.InventoryRepository,
	paymentGateway paymentgateway.Provider,
	auditLogger applogger.AuditLogger,
	transactor transaction.Transactor,
	executor transaction.Executor,
) *ProcessPaymentWebhookUsecase {
	return &ProcessPaymentWebhookUsecase{
		repository:       repository,
		paymentEventRepo: paymentEventRepo,
		webhookEventRepo: webhookEventRepo,
		orderMgr:         orderMgr,
		inventoryRepo:    inventoryRepo,
		paymentGateway:   paymentGateway,
		auditLogger:      auditLogger,
		transactor:       transactor,
		executor:         executor,
	}
}

type ProcessPaymentWebhookInput struct {
	Payload            map[string]any
	NotificationResult *paymentgateway.NotificationResult
}

// Execute processes payment notifications from the gateway webhook or
// reconciliation workflows.
//
// Processing is idempotent and concurrency-safe. Each notification is
// recorded as a unique payment event, while payment state transitions are
// protected against concurrent processing.
//
// Valid notifications update the payment, order, and inventory state according
// to the resolved payment outcome. Settlement commits reserved inventory,
// while cancellation, expiration, or denial releases it.
//
// Business processing failures are persisted on the payment event and returned
// to the delivery handler so the gateway can retry the notification.
func (u *ProcessPaymentWebhookUsecase) Execute(
	ctx context.Context,
	input ProcessPaymentWebhookInput,
) (err error) {
	var (
		orderIDStr string
		txStatus   string
		eventID    uuid.UUID
		markErr    error
	)

	defer func() {
		if err != nil &&
			orderIDStr != "" &&
			eventID != uuid.Nil {

			u.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "user_action",
				Action:     "webhook_processing_failed",
				Resource:   "payment",
				ResourceID: orderIDStr,
				Outcome:    applogger.OutcomeFailure,
				Metadata: map[string]any{
					"error":              err.Error(),
					"transaction_status": txStatus,
					"webhook_event_id":   eventID.String(),
				},
			})
		} else if markErr != nil &&
			orderIDStr != "" &&
			eventID != uuid.Nil {

			u.auditLogger.Log(ctx, applogger.AuditEvent{
				Category:   "user_action",
				Action:     "webhook_mark_processed_failed",
				Resource:   "payment",
				ResourceID: orderIDStr,
				Outcome:    applogger.OutcomeFailure,
				Metadata: map[string]any{
					"error":            markErr.Error(),
					"webhook_event_id": eventID.String(),
				},
			})
		}
	}()

	// Extract idempotency fields
	if input.NotificationResult != nil {
		orderIDStr = input.NotificationResult.GatewayOrderID
		txStatus = input.NotificationResult.RawStatus
	} else if input.Payload != nil {
		orderIDStr, _ = input.Payload["order_id"].(string)
		txStatus, _ = input.Payload["transaction_status"].(string)
	}

	if orderIDStr == "" {
		return apperrors.NewBadRequest("missing order_id in webhook payload")
	}
	if txStatus == "" {
		return apperrors.NewBadRequest("missing transaction_status in webhook payload")
	}

	var payloadBytes []byte
	if input.Payload != nil {
		b, err := json.Marshal(input.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
		payloadBytes = b
	} else if input.NotificationResult != nil {
		b, err := json.Marshal(map[string]any{
			"order_id":           orderIDStr,
			"transaction_status": txStatus,
			"gross_amount":       input.NotificationResult.GrossAmount,
			"source":             "reconciliation",
		})
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
		payloadBytes = b
	}

	// Persist the raw webhook event (idempotency gate)
	//
	// Upsert uses INSERT ... ON CONFLICT DO NOTHING then returns the
	// canonical row.
	//
	// If the same (order_id, transaction_status) has already
	// been successfully processed, system will short-circuit immediately.
	webhookEvent := paymentdomain.PaymentWebhookEvent{
		ID:                uuid.New(),
		OrderID:           orderIDStr,
		TransactionStatus: txStatus,
		Payload:           payloadBytes,
		Status:            paymentdomain.WebhookEventStatusReceived,
		ReceivedAt:        appclock.Now(),
	}

	canonicalEvent, err := u.webhookEventRepo.Upsert(ctx, u.executor,
		webhookEvent,
	)
	if err != nil {
		return fmt.Errorf("failed to persist webhook event: %w", err)
	}

	if canonicalEvent != nil &&
		canonicalEvent.Status == paymentdomain.WebhookEventStatusProcessed {
		return nil
	}

	// Use the canonical event ID for subsequent status updates
	// (it may belong to an earlier delivery attempt, not the one just inserted).
	eventID = canonicalEvent.ID

	// Process the webhook
	//
	// Any error here is recorded on the persisted event and
	// re-surfaced to the caller
	// (non-2xx â†’ Midtrans will retry delivery).
	if err = u.process(ctx, input); err != nil {
		// MarkFailed is intentionally outside the main transaction
		// so it persists even when the inner tx rolls back.
		_ = u.webhookEventRepo.MarkFailed(ctx, u.executor,
			eventID,
			err.Error(),
		)

		return err
	}

	// Stamp the event as processed
	if markErr = u.webhookEventRepo.MarkProcessed(ctx, u.executor,
		eventID,
	); markErr != nil {
		// Non-fatal: the payment itself was updated correctly.
		//
		// Log and continue â€” the event will show as 'received'
		// but won't be re-processed because the payment status
		// is no longer 'pending'.
	}

	return nil
}

// process contains the core payment-state-machine logic.
//
// It is called only after the idempotency gate passes.
func (u *ProcessPaymentWebhookUsecase) process(
	ctx context.Context,
	input ProcessPaymentWebhookInput,
) error {
	var (
		notifResult *paymentgateway.NotificationResult
		err         error
	)

	if input.NotificationResult != nil {
		notifResult = input.NotificationResult
	} else {
		notifResult, err = u.paymentGateway.ParseNotification(ctx, input.Payload)
		if err != nil {
			if errors.Is(err, paymentgateway.ErrInvalidSignature) {
				return apperrors.NewBadRequest("invalid webhook signature")
			}
			return fmt.Errorf("failed to parse gateway notification: %w", err)
		}
	}

	orderID, err := uuid.Parse(notifResult.GatewayOrderID)
	if err != nil {
		return apperrors.NewBadRequest(fmt.Sprintf("invalid order ID in gateway response: %s", notifResult.GatewayOrderID))
	}

	err = u.transactor.WithinTransaction(ctx, func(exec transaction.Executor) error {
		payment, err := u.repository.GetByOrderIDForUpdate(ctx, exec,
			orderID,
		)
		if err != nil {
			return fmt.Errorf("failed to retrieve payment: %w", err)
		}
		if payment == nil {
			return apperrors.NewNotFound("payment not found for order")
		}
		if payment.Status != paymentdomain.PaymentStatusPending {
			return nil
		}

		// Derive payment and order status transitions
		// from webhook input
		//
		// This ensures a strict coupling between
		// payment status and order status: they must always
		// transition together within the same transactional boundary
		var (
			newPaymentStatus paymentdomain.PaymentStatus
			action           string
		)

		switch notifResult.Status {
		case paymentgateway.NotificationStatusSettlement:
			newPaymentStatus = paymentdomain.PaymentStatusPaid
			action = "commit"

		case paymentgateway.NotificationStatusExpire:
			newPaymentStatus = paymentdomain.PaymentStatusExpired
			action = "release_expire"

		case paymentgateway.NotificationStatusCancel:
			newPaymentStatus = paymentdomain.PaymentStatusCancelled
			action = "release_cancel"

		case paymentgateway.NotificationStatusDeny:
			newPaymentStatus = paymentdomain.PaymentStatusFailed
			action = "release_cancel"

		default:
			return nil
		}

		if err := u.repository.UpdateStatus(ctx, exec,
			payment.ID,
			newPaymentStatus,
		); err != nil {
			return fmt.Errorf("failed to update payment status: %w", err)
		}

		var (
			orderItems []OrderItemInfo
			orderErr   error
		)
		now := appclock.Now()
		switch action {
		case "commit":
			orderItems, orderErr = u.orderMgr.ConfirmOrderPayment(ctx, exec, payment.OrderID, now, 72*time.Hour)
		case "release_expire":
			orderItems, orderErr = u.orderMgr.ExpireOrderPayment(ctx, exec, payment.OrderID)
		case "release_cancel":
			orderItems, orderErr = u.orderMgr.CancelOrderPayment(ctx, exec, payment.OrderID)
		}
		if orderErr != nil {
			return orderErr
		}

		for _, item := range orderItems {
			switch action {
			case "commit":
				if err := u.inventoryRepo.Commit(ctx, exec,
					item.ProductID,
					item.ShopID,
					item.Quantity,
				); err != nil {
					return fmt.Errorf("failed to commit inventory for product %s: %w", item.ProductID, err)
				}

			default:
				if err := u.inventoryRepo.Release(ctx, exec,
					item.ProductID,
					item.ShopID,
					item.Quantity,
				); err != nil {
					if errors.Is(err, inventorydomain.ErrInsufficientReserved) ||
						errors.Is(err, apperrors.ErrNotFound) {

						if u.auditLogger != nil {
							u.auditLogger.Log(ctx, applogger.AuditEvent{
								Category:   "system",
								Action:     "inventory_anomaly_detected",
								Resource:   "inventory",
								ResourceID: item.ProductID.String(),
								Outcome:    applogger.OutcomeFailure,
								Metadata: map[string]any{
									"payment_id":    payment.ID.String(),
									"order_id":      payment.OrderID.String(),
									"product_id":    item.ProductID.String(),
									"shop_id":       item.ShopID.String(),
									"requested_qty": item.Quantity,
									"reason":        err.Error(),
								},
							})
						}
						continue
					}
					return fmt.Errorf("failed to release inventory for product %s: %w", item.ProductID, err)
				}
			}
		}

		// Emit payment event for audit
		// and downstream processing
		//
		// This event acts as a durable audit log
		// of the final payment resolution state
		payloadBytes, err := json.Marshal(map[string]any{
			"status":        string(newPaymentStatus),
			"raw_status":    notifResult.RawStatus,
			"fraud_status":  notifResult.FraudStatus,
			"gross_amount":  notifResult.GrossAmount,
			"gateway_tx_id": notifResult.GatewayTransactionID,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal payment event payload: %w", err)
		}

		paymentEvent := paymentdomain.PaymentEvent{
			ID:        uuid.New(),
			PaymentID: payment.ID,
			EventName: string(newPaymentStatus),
			Payload:   payloadBytes,
			CreatedAt: appclock.Now(),
		}

		if err := u.paymentEventRepo.Create(ctx, exec, paymentEvent); err != nil {
			return fmt.Errorf("failed to create payment event: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
