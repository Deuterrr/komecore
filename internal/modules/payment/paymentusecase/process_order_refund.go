package paymentusecase

import (
	"context"
	"fmt"
	"strings"

	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type ProcessOrderRefundUsecase struct {
	paymentRepo    payment.Repository
	paymentGateway paymentgateway.Provider
	executor       transaction.Executor
	transactor     transaction.Transactor
	logger         applogger.Logger
}

func NewProcessOrderRefundUsecase(
	paymentRepo payment.Repository,
	paymentGateway paymentgateway.Provider,
	executor transaction.Executor,
	transactor transaction.Transactor,
	logger applogger.Logger,
) *ProcessOrderRefundUsecase {
	return &ProcessOrderRefundUsecase{
		paymentRepo:    paymentRepo,
		paymentGateway: paymentGateway,
		executor:       executor,
		transactor:     transactor,
		logger:         logger,
	}
}

func (u *ProcessOrderRefundUsecase) Execute(ctx context.Context, orderID uuid.UUID, reason string) error {
	pmt, err := u.paymentRepo.GetByOrderID(ctx, u.executor, orderID)
	if err != nil {
		return fmt.Errorf("failed to get payment for order %s: %w", orderID, err)
	}
	if pmt == nil {
		return fmt.Errorf("payment not found for order %s", orderID)
	}

	if pmt.Status != payment.PaymentStatusPaid {
		u.logger.Info(ctx, "skipping refund as payment is not in paid status",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "payment_status", Value: string(pmt.Status)},
		)
		return nil
	}

	// Determine if automated gateway refund can be attempted
	canAutoRefund := pmt.Provider == "gateway" &&
		pmt.ProviderOrderID != nil &&
		*pmt.ProviderOrderID != ""

	if canAutoRefund {
		refundReq := paymentgateway.RefundRequest{
			GatewayOrderID: *pmt.ProviderOrderID,
			RefundAmount:   pmt.Amount,
			Reason:         reason,
		}

		u.logger.Info(ctx, "attempting automated gateway refund",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "gateway_order_id", Value: *pmt.ProviderOrderID},
		)

		resp, gatewayErr := u.paymentGateway.RefundTransaction(ctx, refundReq)
		if gatewayErr == nil && resp != nil {
			u.logger.Info(ctx, "automated gateway refund successful",
				applogger.Field{Key: "order_id", Value: orderID.String()},
				applogger.Field{Key: "gateway_transaction_id", Value: resp.GatewayTransactionID},
			)
			return u.paymentRepo.UpdateStatus(ctx, u.executor, pmt.ID, payment.PaymentStatusRefunded)
		}

		u.logger.Warn(ctx, "automated gateway refund failed or not supported by channel; flagging for manual refund queue",
			applogger.Field{Key: "order_id", Value: orderID.String()},
			applogger.Field{Key: "error", Value: func() string {
				if gatewayErr != nil {
					return gatewayErr.Error()
				}
				return "unknown gateway refund response"
			}()},
		)
	} else {
		u.logger.Info(ctx, "payment method does not support automated refund; enqueueing for manual refund",
			applogger.Field{Key: "order_id", Value: orderID.String()},
		)
	}

	// Fallback to manual refund queue status
	return u.paymentRepo.UpdateStatus(ctx, u.executor, pmt.ID, payment.PaymentStatusRefundPending)
}

func isAutoRefundableChannel(channel string) bool {
	channelLower := strings.ToLower(channel)
	return strings.Contains(channelLower, "gopay") ||
		strings.Contains(channelLower, "shopeepay") ||
		strings.Contains(channelLower, "qris") ||
		strings.Contains(channelLower, "credit_card")
}
