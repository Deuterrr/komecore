package paymentusecase

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"

	"github.com/google/uuid"
)

type refundMockPaymentRepo struct {
	payment *paymentdomain.Payment
	status  paymentdomain.PaymentStatus
}

func (m *refundMockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.Payment, error) {
	return nil, nil
}
func (m *refundMockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.Payment, error) {
	return m.payment, nil
}
func (m *refundMockPaymentRepo) GetByOrderIDForUpdate(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentdomain.Payment, error) {
	return m.payment, nil
}
func (m *refundMockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]paymentdomain.Payment, error) {
	return nil, nil
}
func (m *refundMockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, _ uuid.UUID, status paymentdomain.PaymentStatus) error {
	m.status = status
	return nil
}
func (m *refundMockPaymentRepo) Save(_ context.Context, _ transaction.Executor, _ paymentdomain.Payment) error {
	return nil
}
func (m *refundMockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]paymentdomain.Payment, error) {
	return nil, nil
}
func (m *refundMockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]paymentdomain.Payment, error) {
	return nil, nil
}

type refundMockGateway struct{}

func (m *refundMockGateway) Name() string                                                 { return "mock" }
func (m *refundMockGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod { return nil }
func (m *refundMockGateway) Supports(code string) bool                                    { return true }
func (m *refundMockGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return nil, nil
}
func (m *refundMockGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (m *refundMockGateway) GetTransactionStatus(_ context.Context, _ string) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (m *refundMockGateway) CancelTransaction(_ context.Context, _ string) error { return nil }
func (m *refundMockGateway) RefundTransaction(_ context.Context, req paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return &paymentgateway.RefundResponse{
		GatewayTransactionID: "tx-123",
		GatewayOrderID:       req.GatewayOrderID,
		RefundAmount:         req.RefundAmount,
		Status:               "200",
	}, nil
}

func TestProcessOrderRefundUsecase_NonPaidPayment(t *testing.T) {
	orderID := uuid.New()
	repo := &refundMockPaymentRepo{
		payment: &paymentdomain.Payment{
			ID:      uuid.New(),
			OrderID: orderID,
			Status:  paymentdomain.PaymentStatusPending,
		},
	}
	gw := &refundMockGateway{}
	exec := &mockExecutor{}
	trans := &mockTransactor{}
	logger := &mockLogger{}

	uc := NewProcessOrderRefundUsecase(repo, gw, exec, trans, logger)

	err := uc.Execute(context.Background(), orderID, "test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.status != "" {
		t.Fatalf("expected status not to change for pending payment, got %s", repo.status)
	}
}

func TestProcessOrderRefundUsecase_PaidPaymentAutomatedRefund(t *testing.T) {
	orderID := uuid.New()
	providerOrderID := "ord-123"
	repo := &refundMockPaymentRepo{
		payment: &paymentdomain.Payment{
			ID:              uuid.New(),
			OrderID:         orderID,
			Provider:        "gateway",
			ProviderOrderID: &providerOrderID,
			Amount:          50000,
			Status:          paymentdomain.PaymentStatusPaid,
		},
	}
	gw := &refundMockGateway{}
	exec := &mockExecutor{}
	trans := &mockTransactor{}
	logger := &mockLogger{}

	uc := NewProcessOrderRefundUsecase(repo, gw, exec, trans, logger)

	err := uc.Execute(context.Background(), orderID, "staff SLA expired")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.status != paymentdomain.PaymentStatusRefunded {
		t.Fatalf("expected status to be refunded, got %s", repo.status)
	}
}
