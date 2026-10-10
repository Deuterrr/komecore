package paymentusecase

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"
	"komecore/internal/modules/payment/paymentdomain"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type mockLogger struct {
	infos  []string
	errors []string
	warns  []string
	debugs []string
}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...applogger.Field) {
	m.infos = append(m.infos, msg)
}

func (m *mockLogger) Error(_ context.Context, msg string, _ ...applogger.Field) {
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) Warn(_ context.Context, msg string, _ ...applogger.Field) {
	m.warns = append(m.warns, msg)
}

func (m *mockLogger) Debug(_ context.Context, msg string, _ ...applogger.Field) {
	m.debugs = append(m.debugs, msg)
}

func (m *mockLogger) With(_ ...applogger.Field) applogger.Logger {
	return m
}

func TestSyncPendingPayments_ReconcileOnePaid(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:     orderID,
		Status: "pending",
	}

	providerOrderID := orderID.String()
	payment := &paymentdomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentdomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{paymentID: payment}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID:       orderID.String(),
			GatewayTransactionID: "tx-123",
			Status:               paymentgateway.NotificationStatusSettlement,
			RawStatus:            "settlement",
			GrossAmount:          100000,
		},
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	usecase := NewSyncPendingPaymentsUsecase(pRepo, gateway, webhookUsecase, &mockExecutor{}, logger, 24*time.Hour, &mockTransactor{}, oMgr, &mockInventoryRepo{})

	usecase.Execute(ctx)

	if payment.Status != paymentdomain.PaymentStatusPaid {
		t.Errorf("expected payment status to be updated to Paid, got %v", payment.Status)
	}

	if order.Status != "confirmed" {
		t.Errorf("expected order status to be Confirmed, got %v", order.Status)
	}
}

func TestSyncPendingPayments_SkipStillPending(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:     orderID,
		Status: "pending",
	}

	providerOrderID := orderID.String()
	payment := &paymentdomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentdomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{paymentID: payment}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusPending,
			RawStatus:      "pending",
		},
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	usecase := NewSyncPendingPaymentsUsecase(pRepo, gateway, webhookUsecase, &mockExecutor{}, logger, 24*time.Hour, &mockTransactor{}, oMgr, &mockInventoryRepo{})

	usecase.Execute(ctx)

	if payment.Status != paymentdomain.PaymentStatusPending {
		t.Errorf("expected payment status to remain Pending, got %v", payment.Status)
	}
}

type customTestGateway struct {
	results map[string]*paymentgateway.NotificationResult
	errs    map[string]error
}

func (c *customTestGateway) Name() string                                                 { return "custom_test_gateway" }
func (c *customTestGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod { return nil }
func (c *customTestGateway) Supports(_ string) bool                                       { return true }
func (c *customTestGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return nil, nil
}
func (c *customTestGateway) ParseNotification(_ context.Context, payload paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	orderID, _ := payload["order_id"].(string)
	if err, ok := c.errs[orderID]; ok && err != nil {
		return nil, err
	}
	return c.results[orderID], nil
}
func (c *customTestGateway) GetTransactionStatus(_ context.Context, gatewayOrderID string) (*paymentgateway.NotificationResult, error) {
	if err, ok := c.errs[gatewayOrderID]; ok && err != nil {
		return nil, err
	}
	return c.results[gatewayOrderID], nil
}
func (c *customTestGateway) CancelTransaction(_ context.Context, _ string) error { return nil }
func (c *customTestGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}

func TestSyncPendingPayments_GatewayErrorDoesNotBlockOtherPayments(t *testing.T) {
	ctx := context.Background()

	orderID1 := uuid.New()
	paymentID1 := uuid.New()
	orderID2 := uuid.New()
	paymentID2 := uuid.New()

	order1 := &mockOrder{ID: orderID1, Status: "pending"}
	order2 := &mockOrder{ID: orderID2, Status: "pending"}

	providerOrderID1 := orderID1.String()
	providerOrderID2 := orderID2.String()

	payment1 := &paymentdomain.Payment{
		ID:              paymentID1,
		OrderID:         orderID1,
		Status:          paymentdomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID1,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}
	payment2 := &paymentdomain.Payment{
		ID:              paymentID2,
		OrderID:         orderID2,
		Status:          paymentdomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID2,
		Amount:          200000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{
		paymentID1: payment1,
		paymentID2: payment2,
	}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{
		orderID1: order1,
		orderID2: order2,
	}}

	gateway := &customTestGateway{
		results: map[string]*paymentgateway.NotificationResult{
			providerOrderID2: {
				GatewayOrderID:       orderID2.String(),
				GatewayTransactionID: "tx-456",
				Status:               paymentgateway.NotificationStatusSettlement,
				RawStatus:            "settlement",
				GrossAmount:          200000,
			},
		},
		errs: map[string]error{
			providerOrderID1: assertError("gateway connectivity timeout"),
		},
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	usecase := NewSyncPendingPaymentsUsecase(pRepo, gateway, webhookUsecase, &mockExecutor{}, logger, 24*time.Hour, &mockTransactor{}, oMgr, &mockInventoryRepo{})

	usecase.Execute(ctx)

	// Payment 1 should remain pending (due to error)
	if payment1.Status != paymentdomain.PaymentStatusPending {
		t.Errorf("expected payment1 status to remain Pending on error, got %v", payment1.Status)
	}

	// Payment 2 should be processed and paid
	if payment2.Status != paymentdomain.PaymentStatusPaid {
		t.Errorf("expected payment2 status to be Paid, got %v", payment2.Status)
	}

	if len(logger.errors) != 1 {
		t.Errorf("expected 1 error logged, got %d", len(logger.errors))
	}
}

type gatewayErr struct {
	msg string
}

func (e gatewayErr) Error() string { return e.msg }
func assertError(msg string) error {
	return gatewayErr{msg: msg}
}

func TestSyncPendingPayments_EnforceLocalExpiry(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:     orderID,
		Status: "pending",
	}

	providerOrderID := orderID.String()
	pastTime := time.Now().Add(-1 * time.Hour)
	payment := &paymentdomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentdomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		ExpiresAt:       &pastTime,
		CreatedAt:       time.Now().Add(-2 * time.Hour),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{paymentID: payment}}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items: map[uuid.UUID][]OrderItemInfo{
			orderID: {
				{
					ProductID: uuid.New(),
					ShopID:    uuid.New(),
					Quantity:  2,
				},
			},
		},
	}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusPending,
			RawStatus:      "pending",
		},
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil)
	logger := &mockLogger{}
	usecase := NewSyncPendingPaymentsUsecase(pRepo, gateway, webhookUsecase, &mockExecutor{}, logger, 24*time.Hour, &mockTransactor{}, oMgr, iRepo)

	usecase.Execute(ctx)

	if payment.Status != paymentdomain.PaymentStatusExpired {
		t.Errorf("expected payment status to be updated to Expired, got %v", payment.Status)
	}

	if order.Status != "expired" {
		t.Errorf("expected order status to be Expired, got %v", order.Status)
	}

	// Verify that inventory was released
	if len(iRepo.releases) != 1 {
		t.Errorf("expected 1 inventory release call, got %d", len(iRepo.releases))
	}
}
