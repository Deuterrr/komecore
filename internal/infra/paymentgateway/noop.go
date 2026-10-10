package paymentgateway

import (
	"context"
	"time"
)

// NoopProvider provides a safe, no-operation fallback implementation of Provider
// when no external payment gateway is configured or for testing/offline environments.
type NoopProvider struct{}

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

func (n *NoopProvider) Name() string {
	return "noop"
}

func (n *NoopProvider) AllowedPaymentMethods() []AllowedPaymentMethod {
	return []AllowedPaymentMethod{
		{
			Code:        "manual_transfer",
			Name:        "Manual Transfer",
			Type:        "bank_transfer",
			FeeType:     "flat",
			FeeFixed:    0,
			Description: "Manual offline bank transfer",
		},
	}
}

func (n *NoopProvider) Supports(code string) bool {
	return true
}

func (n *NoopProvider) Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	txID := "noop-tx-" + req.PaymentID.String()
	return &ChargeResponse{
		GatewayTransactionID: txID,
		GatewayOrderID:       req.PaymentID.String(),
		PaymentType:          req.PaymentType,
		GrossAmount:          req.Amount,
		Status:               "pending",
		ExpiresAt:            time.Now().Add(24 * time.Hour),
	}, nil
}

func (n *NoopProvider) ParseNotification(ctx context.Context, payload NotificationPayload) (*NotificationResult, error) {
	return &NotificationResult{
		GatewayTransactionID: "noop-tx",
		GatewayOrderID:       "noop-order",
		Status:               NotificationStatusSettlement,
		GrossAmount:          0,
		RawStatus:            "settlement",
	}, nil
}

func (n *NoopProvider) GetTransactionStatus(ctx context.Context, gatewayOrderID string) (*NotificationResult, error) {
	return &NotificationResult{
		GatewayTransactionID: "noop-tx-" + gatewayOrderID,
		GatewayOrderID:       gatewayOrderID,
		Status:               NotificationStatusSettlement,
		GrossAmount:          0,
		RawStatus:            "settlement",
	}, nil
}

func (n *NoopProvider) CancelTransaction(ctx context.Context, gatewayOrderID string) error {
	return nil
}

func (n *NoopProvider) RefundTransaction(ctx context.Context, req RefundRequest) (*RefundResponse, error) {
	return &RefundResponse{
		GatewayTransactionID: "noop-refund-" + req.GatewayOrderID,
		GatewayOrderID:       req.GatewayOrderID,
		RefundAmount:         req.RefundAmount,
		Status:               "refunded",
	}, nil
}
