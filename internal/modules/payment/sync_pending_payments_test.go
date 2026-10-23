package payment

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

type mockLogger struct {
	mu     sync.Mutex
	infos  []string
	errors []string
	warns  []string
	debugs []string
}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...applogger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos = append(m.infos, msg)
}

func (m *mockLogger) Error(_ context.Context, msg string, _ ...applogger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, msg)
}

func (m *mockLogger) Warn(_ context.Context, msg string, _ ...applogger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warns = append(m.warns, msg)
}

func (m *mockLogger) Debug(_ context.Context, msg string, _ ...applogger.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugs = append(m.debugs, msg)
}

func (m *mockLogger) With(_ ...applogger.Field) applogger.Logger {
	return m
}

func (m *mockLogger) errorCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.errors)
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
	pmt := &Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
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

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, 5)

	svc.SyncPendingPayments(ctx)

	if pmt.Status != PaymentStatusPaid {
		t.Errorf("expected payment status to be updated to Paid, got %v", pmt.Status)
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
	pmt := &Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusPending,
			RawStatus:      "pending",
		},
	}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, 5)

	svc.SyncPendingPayments(ctx)

	if pmt.Status != PaymentStatusPending {
		t.Errorf("expected payment status to remain Pending, got %v", pmt.Status)
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

	payment1 := &Payment{
		ID:              paymentID1,
		OrderID:         orderID1,
		Status:          PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID1,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}
	payment2 := &Payment{
		ID:              paymentID2,
		OrderID:         orderID2,
		Status:          PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID2,
		Amount:          200000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{
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

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, 5)

	svc.SyncPendingPayments(ctx)

	// Payment 1 should remain pending (due to error)
	if payment1.Status != PaymentStatusPending {
		t.Errorf("expected payment1 status to remain Pending on error, got %v", payment1.Status)
	}

	// Payment 2 should be processed and paid
	if payment2.Status != PaymentStatusPaid {
		t.Errorf("expected payment2 status to be Paid, got %v", payment2.Status)
	}

	if logger.errorCount() != 1 {
		t.Errorf("expected 1 error logged, got %d", logger.errorCount())
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
	pmt := &Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		ExpiresAt:       &pastTime,
		CreatedAt:       time.Now().Add(-2 * time.Hour),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
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

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, iRepo, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, 5)

	svc.SyncPendingPayments(ctx)

	if pmt.Status != PaymentStatusExpired {
		t.Errorf("expected payment status to be updated to Expired, got %v", pmt.Status)
	}

	if order.Status != "expired" {
		t.Errorf("expected order status to be Expired, got %v", order.Status)
	}

	// Verify that inventory was released
	if len(iRepo.releases) != 1 {
		t.Errorf("expected 1 inventory release call, got %d", len(iRepo.releases))
	}
}

type parallelTestGateway struct {
	latency         time.Duration
	currentInFlight int32
	maxInFlight     int32
}

func (p *parallelTestGateway) Name() string { return "parallel_test_gateway" }
func (p *parallelTestGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod {
	return nil
}
func (p *parallelTestGateway) Supports(_ string) bool { return true }
func (p *parallelTestGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return nil, nil
}
func (p *parallelTestGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return nil, nil
}
func (p *parallelTestGateway) GetTransactionStatus(_ context.Context, gatewayOrderID string) (*paymentgateway.NotificationResult, error) {
	cur := atomic.AddInt32(&p.currentInFlight, 1)
	for {
		oldMax := atomic.LoadInt32(&p.maxInFlight)
		if cur <= oldMax || atomic.CompareAndSwapInt32(&p.maxInFlight, oldMax, cur) {
			break
		}
	}
	if p.latency > 0 {
		time.Sleep(p.latency)
	}
	atomic.AddInt32(&p.currentInFlight, -1)

	return &paymentgateway.NotificationResult{
		GatewayOrderID:       gatewayOrderID,
		GatewayTransactionID: "tx-" + gatewayOrderID,
		Status:               paymentgateway.NotificationStatusSettlement,
		RawStatus:            "settlement",
		GrossAmount:          100000,
	}, nil
}
func (p *parallelTestGateway) CancelTransaction(_ context.Context, _ string) error { return nil }
func (p *parallelTestGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}

func TestSyncPendingPayments_ParallelExecution(t *testing.T) {
	ctx := context.Background()
	const numPayments = 20
	const concurrency = 5
	const latency = 25 * time.Millisecond

	pRepo := &mockPaymentRepo{payments: make(map[uuid.UUID]*Payment)}
	oMgr := newMockOrderPaymentManager()

	for i := 0; i < numPayments; i++ {
		orderID := uuid.New()
		paymentID := uuid.New()
		order := &mockOrder{
			ID:     orderID,
			Status: "pending",
		}
		oMgr.orders[orderID] = order

		providerOrderID := orderID.String()
		pmt := &Payment{
			ID:              paymentID,
			OrderID:         orderID,
			Status:          PaymentStatusPending,
			Provider:        "gateway",
			ProviderOrderID: &providerOrderID,
			Amount:          100000,
			CreatedAt:       time.Now(),
		}
		pRepo.payments[paymentID] = pmt
	}

	gateway := &parallelTestGateway{
		latency: latency,
	}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, concurrency)

	start := time.Now()
	svc.SyncPendingPayments(ctx)
	elapsed := time.Since(start)

	// 1. Verify true parallel execution: maxInFlight must be > 1 (expected up to 5)
	maxConcurrent := atomic.LoadInt32(&gateway.maxInFlight)
	if maxConcurrent <= 1 {
		t.Errorf("expected concurrent gateway requests > 1, got %d", maxConcurrent)
	}

	// 2. Sequential execution time would be 20 * 25ms = 500ms.
	// With 5 workers, 20 / 5 = 4 batches * 25ms = 100ms.
	// We assert elapsed is significantly less than sequential time (< 400ms).
	sequentialThreshold := numPayments * latency
	if elapsed >= sequentialThreshold {
		t.Errorf("expected parallel reconciliation time (%v) to be less than sequential (%v)", elapsed, sequentialThreshold)
	}

	// 3. Verify all payments are marked Paid
	pRepo.mu.RLock()
	for id, pmt := range pRepo.payments {
		if pmt.Status != PaymentStatusPaid {
			t.Errorf("expected payment %s to be Paid, got %v", id, pmt.Status)
		}
	}
	pRepo.mu.RUnlock()

	// 4. Verify all orders are confirmed
	oMgr.mu.RLock()
	for id, order := range oMgr.orders {
		if order.Status != "confirmed" {
			t.Errorf("expected order %s to be confirmed, got %v", id, order.Status)
		}
	}
	oMgr.mu.RUnlock()
}

func TestSyncPendingPayments_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	orderID := uuid.New()
	paymentID := uuid.New()
	providerOrderID := orderID.String()

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{
		paymentID: {
			ID:              paymentID,
			OrderID:         orderID,
			Status:          PaymentStatusPending,
			Provider:        "gateway",
			ProviderOrderID: &providerOrderID,
			Amount:          100000,
			CreatedAt:       time.Now(),
		},
	}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{
		orderID: {ID: orderID, Status: "pending"},
	}}
	gateway := &mockPaymentGateway{}
	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	logger := &mockLogger{}
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, logger, 24*time.Hour, 100, 5)

	// Must not hang or panic
	svc.SyncPendingPayments(ctx)
}
