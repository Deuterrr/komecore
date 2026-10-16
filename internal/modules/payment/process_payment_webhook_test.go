package payment

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"komecore/internal/apperror"
	"komecore/internal/infra/paymentgateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"
	applogger "komecore/pkg/logger"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shared mock infrastructure
// ---------------------------------------------------------------------------

type mockExecutor struct {
	transaction.NoopExecutor
}

type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(transaction.Executor) error) error {
	return fn(&mockExecutor{})
}

// mockPaymentRepo supports configurable updateErr so tests can inject failures.
type mockPaymentRepo struct {
	payments  map[uuid.UUID]*Payment
	updateErr error
}

func (m *mockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*Payment, error) {
	return m.payments[id], nil
}
func (m *mockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	for _, p := range m.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, nil
}
func (m *mockPaymentRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}
func (m *mockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, orderIDs []uuid.UUID) ([]Payment, error) {
	var result []Payment
	idx := make(map[uuid.UUID]bool)
	for _, id := range orderIDs {
		idx[id] = true
	}
	for _, p := range m.payments {
		if idx[p.OrderID] {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (m *mockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status PaymentStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if p, ok := m.payments[id]; ok {
		p.Status = status
		return nil
	}
	return errors.New("not found")
}
func (m *mockPaymentRepo) Save(_ context.Context, _ transaction.Executor, pmt Payment) error {
	m.payments[pmt.ID] = &pmt
	return nil
}
func (m *mockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, since time.Time) ([]Payment, error) {
	var result []Payment
	for _, p := range m.payments {
		if p.Status == PaymentStatusPending &&
			p.Provider == "gateway" &&
			p.ProviderOrderID != nil &&
			(p.CreatedAt.After(since) || p.CreatedAt.Equal(since)) {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (m *mockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]Payment, error) {
	return nil, nil
}

// mockPaymentAccountRepo tracks increments, decrements and supports injecting errors.
type mockPaymentAccountRepo struct{}

// mockPaymentEventRepo tracks created events and supports injecting createErr.
type mockPaymentEventRepo struct {
	events    []PaymentEvent
	createErr error
}

func (m *mockPaymentEventRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*PaymentEvent, error) {
	return nil, nil
}
func (m *mockPaymentEventRepo) ListByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]PaymentEvent, error) {
	return nil, nil
}
func (m *mockPaymentEventRepo) Create(_ context.Context, _ transaction.Executor, event PaymentEvent) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.events = append(m.events, event)
	return nil
}

// mockInventoryRepo supports configurable commitErr, releaseErr, reserveErr.
type mockInventoryRepo struct {
	commits      []string
	releases     []string
	reserveCalls []string
	commitErr    error
	releaseErr   error
	reserveErr   error
}

func (m *mockInventoryRepo) GetByProductIDAndShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) (*inventory.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventory.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) Create(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *mockInventoryRepo) Reserve(_ context.Context, _ transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error {
	m.reserveCalls = append(m.reserveCalls, fmt.Sprintf("%s-%s-%d", productID, shopID, qty))
	return m.reserveErr
}
func (m *mockInventoryRepo) Release(_ context.Context, _ transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error {
	if m.releaseErr != nil {
		return m.releaseErr
	}
	m.releases = append(m.releases, fmt.Sprintf("%s-%s-%d", productID, shopID, qty))
	return nil
}
func (m *mockInventoryRepo) Commit(_ context.Context, _ transaction.Executor, productID uuid.UUID, shopID uuid.UUID, qty int) error {
	if m.commitErr != nil {
		return m.commitErr
	}
	m.commits = append(m.commits, fmt.Sprintf("%s-%s-%d", productID, shopID, qty))
	return nil
}
func (m *mockInventoryRepo) Restock(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockInventoryRepo) Update(_ context.Context, _ transaction.Executor, _ *inventory.Inventory) error {
	return nil
}
func (m *mockInventoryRepo) Delete(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

// mockPaymentGateway for webhook tests (parse notification path).
type mockPaymentGateway struct {
	result *paymentgateway.NotificationResult
	err    error
}

func (m *mockPaymentGateway) Name() string { return "mock_gateway" }
func (m *mockPaymentGateway) AllowedPaymentMethods() []paymentgateway.AllowedPaymentMethod {
	return nil
}
func (m *mockPaymentGateway) Charge(_ context.Context, _ paymentgateway.ChargeRequest) (*paymentgateway.ChargeResponse, error) {
	return nil, nil
}
func (m *mockPaymentGateway) ParseNotification(_ context.Context, _ paymentgateway.NotificationPayload) (*paymentgateway.NotificationResult, error) {
	return m.result, m.err
}
func (m *mockPaymentGateway) GetTransactionStatus(_ context.Context, _ string) (*paymentgateway.NotificationResult, error) {
	return m.result, m.err
}
func (m *mockPaymentGateway) CancelTransaction(_ context.Context, _ string) error {
	return nil
}
func (m *mockPaymentGateway) RefundTransaction(_ context.Context, _ paymentgateway.RefundRequest) (*paymentgateway.RefundResponse, error) {
	return nil, nil
}
func (m *mockPaymentGateway) Supports(_ string) bool {
	return true
}

// mockPaymentWebhookEventRepo is a simple in-memory implementation of
// PaymentWebhookEventRepository used in tests.
type mockPaymentWebhookEventRepo struct {
	events      map[string]*PaymentWebhookEvent // key: order_id+":"+tx_status
	upsertErr   error
	markProcErr error
	markFailErr error
	processed   []uuid.UUID
	failed      []uuid.UUID
}

func newMockWebhookEventRepo() *mockPaymentWebhookEventRepo {
	return &mockPaymentWebhookEventRepo{
		events: make(map[string]*PaymentWebhookEvent),
	}
}

func (m *mockPaymentWebhookEventRepo) Upsert(
	_ context.Context,
	_ transaction.Executor,
	event PaymentWebhookEvent,
) (*PaymentWebhookEvent, error) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	key := event.OrderID + ":" + event.TransactionStatus
	if existing, ok := m.events[key]; ok {
		return existing, nil
	}
	m.events[key] = &event
	return &event, nil
}

func (m *mockPaymentWebhookEventRepo) MarkProcessed(
	_ context.Context,
	_ transaction.Executor,
	id uuid.UUID,
) error {
	if m.markProcErr != nil {
		return m.markProcErr
	}
	m.processed = append(m.processed, id)
	return nil
}

func (m *mockPaymentWebhookEventRepo) MarkFailed(
	_ context.Context,
	_ transaction.Executor,
	id uuid.UUID,
	_ string,
) error {
	if m.markFailErr != nil {
		return m.markFailErr
	}
	m.failed = append(m.failed, id)
	return nil
}

// mockAuditLogger captures audit events emitted during tests.
type mockAuditLogger struct {
	events []applogger.AuditEvent
}

func (m *mockAuditLogger) Log(_ context.Context, event applogger.AuditEvent) {
	m.events = append(m.events, event)
}

// ---------------------------------------------------------------------------
// Helper: assemble a webhook processor with sensible defaults.
// ---------------------------------------------------------------------------

func newWebhookProcessor(
	pRepo *mockPaymentRepo,
	paRepo *mockPaymentAccountRepo,
	peRepo *mockPaymentEventRepo,
	oMgr OrderPaymentManager,
	iRepo *mockInventoryRepo,
	gateway paymentgateway.Provider,
	transactor transaction.Transactor,
) *WebhookProcessor {
	if transactor == nil {
		transactor = &mockTransactor{}
	}
	return NewWebhookProcessor(
		pRepo, peRepo,
		newMockWebhookEventRepo(),
		oMgr, iRepo,
		gateway,
		&mockAuditLogger{},
		transactor, &mockExecutor{},
	)
}

var newWebhookUsecase = newWebhookProcessor

// ---------------------------------------------------------------------------
// All terminal states
// ---------------------------------------------------------------------------

func TestProcessPaymentWebhook_Settlement(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Status:    PaymentStatusPending,
		Amount:    100000,
		Provider:  "midtrans",
		CreatedAt: time.Now(),
	}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 2}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	paRepo := &mockPaymentAccountRepo{}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID:       orderID.String(),
			GatewayTransactionID: "midtrans-tx-123",
			Status:               paymentgateway.NotificationStatusSettlement,
			RawStatus:            "settlement",
			GrossAmount:          100000,
		},
	}

	err := newWebhookProcessor(pRepo, paRepo, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusPaid {
		t.Errorf("expected payment paid, got %v", pmt.Status)
	}
	if order.Status != "confirmed" {
		t.Errorf("expected order confirmed, got %v", order.Status)
	}
	expectedCommit := fmt.Sprintf("%s-%s-2", productID, shopID)
	if len(iRepo.commits) != 1 || iRepo.commits[0] != expectedCommit {
		t.Errorf("expected commits %v, got %v", []string{expectedCommit}, iRepo.commits)
	}
	if len(peRepo.events) != 1 || peRepo.events[0].EventName != "paid" {
		t.Errorf("expected 1 paid event, got %v", peRepo.events)
	}
}

func TestProcessPaymentWebhook_Expire(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Status:    PaymentStatusPending,
		Amount:    100000,
		Provider:  "midtrans",
		CreatedAt: time.Now(),
	}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 2}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	paRepo := &mockPaymentAccountRepo{}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusExpire,
			RawStatus:      "expire",
			GrossAmount:    100000,
		},
	}

	err := newWebhookProcessor(pRepo, paRepo, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "expire"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusExpired {
		t.Errorf("expected payment expired, got %v", pmt.Status)
	}
	if order.Status != "expired" {
		t.Errorf("expected order expired, got %v", order.Status)
	}
	expectedRelease := fmt.Sprintf("%s-%s-2", productID, shopID)
	if len(iRepo.releases) != 1 || iRepo.releases[0] != expectedRelease {
		t.Errorf("expected releases %v, got %v", []string{expectedRelease}, iRepo.releases)
	}
	if len(peRepo.events) != 1 || peRepo.events[0].EventName != "expired" {
		t.Errorf("expected 1 expired event, got %v", peRepo.events)
	}
}

func TestProcessPaymentWebhook_Cancel(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
			RawStatus:      "cancel",
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusCancelled {
		t.Errorf("expected payment cancelled, got %v", pmt.Status)
	}
	if order.Status != "cancelled" {
		t.Errorf("expected order cancelled, got %v", order.Status)
	}
	if len(iRepo.releases) != 1 {
		t.Errorf("expected 1 inventory release, got %d", len(iRepo.releases))
	}
}

func TestProcessPaymentWebhook_Deny(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 3}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusDeny,
			RawStatus:      "deny",
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "deny"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusFailed {
		t.Errorf("expected payment failed, got %v", pmt.Status)
	}
	if order.Status != "cancelled" {
		t.Errorf("expected order cancelled, got %v", order.Status)
	}
	if len(iRepo.releases) != 1 {
		t.Errorf("expected 1 inventory release for denied payment, got %d", len(iRepo.releases))
	}
}

// ---------------------------------------------------------------------------
// Pending / challenge are NOT terminal — must be no-ops
// ---------------------------------------------------------------------------

func TestProcessPaymentWebhook_Pending_IsNoOp(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 50000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusPending,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "pending"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusPending {
		t.Errorf("payment status changed unexpectedly for pending: %v", pmt.Status)
	}
	if len(peRepo.events) != 0 {
		t.Errorf("expected no events for pending notification, got %d", len(peRepo.events))
	}
	if len(iRepo.commits) != 0 || len(iRepo.releases) != 0 {
		t.Error("inventory should not have been touched for pending notification")
	}
}

func TestProcessPaymentWebhook_Challenge_IsNoOp(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 50000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusChallenge,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "challenge"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pmt.Status != PaymentStatusPending {
		t.Errorf("payment status changed unexpectedly for challenge: %v", pmt.Status)
	}
	if len(peRepo.events) != 0 {
		t.Errorf("expected no events for challenge notification, got %d", len(peRepo.events))
	}
}

// ---------------------------------------------------------------------------
// Idempotency: duplicate webhooks after final state — must be silent no-ops
// ---------------------------------------------------------------------------

func TestProcessPaymentWebhook_Idempotent_AlreadyPaid(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPaid, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "confirmed"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peRepo.events) != 0 {
		t.Errorf("expected no new events for already-paid payment, got %d", len(peRepo.events))
	}
	if len(iRepo.commits) != 0 {
		t.Error("inventory must not be committed again for already-paid payment")
	}
}

func TestProcessPaymentWebhook_Idempotent_AlreadyCancelled(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusCancelled, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "cancelled"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, peRepo, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peRepo.events) != 0 {
		t.Errorf("expected no events for already-cancelled payment, got %d", len(peRepo.events))
	}
	if len(iRepo.releases) != 0 {
		t.Error("inventory must not be released again for already-cancelled payment")
	}
}

// ---------------------------------------------------------------------------
// Coordination protocol / failure recovery
// ---------------------------------------------------------------------------

func TestProcessPaymentWebhook_PaymentNotFound(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{}},
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})

	if err == nil {
		t.Fatal("expected error when payment not found, got nil")
	}
}

func TestProcessPaymentWebhook_InvalidOrderIDInGatewayResponse(t *testing.T) {
	ctx := context.Background()
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: "not-a-uuid",
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(&mockPaymentRepo{payments: map[uuid.UUID]*Payment{}},
		&mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{}},
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": "not-a-uuid", "transaction_status": "settlement"})

	if err == nil {
		t.Fatal("expected error for invalid order ID in gateway response")
	}
}

func TestProcessPaymentWebhook_GatewayParseError(t *testing.T) {
	ctx := context.Background()
	gateway := &mockPaymentGateway{err: errors.New("midtrans: gateway unreachable")}

	err := newWebhookProcessor(&mockPaymentRepo{payments: map[uuid.UUID]*Payment{}},
		&mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{}},
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": uuid.New().String(), "transaction_status": "settlement"})

	if err == nil {
		t.Fatal("expected error when gateway.ParseNotification fails")
	}
}

func TestProcessPaymentWebhook_PaymentRepoUpdateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	pRepo := &mockPaymentRepo{
		payments:  map[uuid.UUID]*Payment{paymentID: pmt},
		updateErr: errors.New("db: update failed"),
	}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr,
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})

	if err == nil {
		t.Fatal("expected error when payment repo update fails")
	}
}

func TestProcessPaymentWebhook_OrderRepoUpdateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{
		orders:    map[uuid.UUID]*mockOrder{orderID: order},
		updateErr: errors.New("db: order update failed"),
	}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr,
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})

	if err == nil {
		t.Fatal("expected error when order repo update fails")
	}
}

func TestProcessPaymentWebhook_InventoryCommitFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{commitErr: errors.New("inventory: stock underflow")}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})
	if err == nil {
		t.Fatal("expected error when inventory commit fails")
	}
}

func TestProcessPaymentWebhook_InventoryReleaseFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	iRepo := &mockInventoryRepo{releaseErr: errors.New("inventory: release error")}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusExpire,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, iRepo, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "expire"})
	if err == nil {
		t.Fatal("expected error when inventory release fails")
	}
}

func TestProcessPaymentWebhook_PaymentEventCreateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &mockOrder{ID: orderID, Status: "pending"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	peRepo := &mockPaymentEventRepo{createErr: errors.New("db: event insert failed")}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, peRepo, oMgr,
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"})
	if err == nil {
		t.Fatal("expected error when payment event creation fails")
	}
}

func TestProcessPaymentWebhook_InvalidOrderStatusTransition(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pmt := &Payment{ID: paymentID, OrderID: orderID, Status: PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	// Order is already delivered, so transition to cancelled is invalid
	order := &mockOrder{ID: orderID, Status: "delivered"}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
		},
	}

	err := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr,
		&mockInventoryRepo{}, gateway, nil).
		ProcessWebhook(ctx, map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"})
	if err == nil {
		t.Fatal("expected error for invalid order status transition via webhook, got nil")
	}
}

func TestProcessPaymentWebhook_DirectNotificationResult_BypassesParseNotification(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pRepo := &mockPaymentRepo{
		payments: map[uuid.UUID]*Payment{
			paymentID: {
				ID:      paymentID,
				OrderID: orderID,
				Status:  PaymentStatusPending,
				Amount:  100000,
			},
		},
	}
	peRepo := &mockPaymentEventRepo{}
	oMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{
			orderID: {
				ID:     orderID,
				Status: "pending",
			},
		},
	}
	invRepo := &mockInventoryRepo{}
	// gateway with error if ParseNotification is called
	gateway := &mockPaymentGateway{
		err: errors.New("should not be called"),
	}

	uc := NewWebhookProcessor(
		pRepo, peRepo,
		newMockWebhookEventRepo(),
		oMgr,
		invRepo,
		gateway,
		&mockAuditLogger{},
		&mockTransactor{}, &mockExecutor{},
	)

	err := uc.Execute(ctx, ProcessPaymentWebhookInput{
		NotificationResult: &paymentgateway.NotificationResult{
			GatewayOrderID:       orderID.String(),
			GatewayTransactionID: "tx-direct-123",
			Status:               paymentgateway.NotificationStatusSettlement,
			GrossAmount:          100000,
			RawStatus:            "settlement",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error when executing with direct NotificationResult: %v", err)
	}

	// Verify payment is paid
	p := pRepo.payments[paymentID]
	if p.Status != PaymentStatusPaid {
		t.Errorf("payment status = %v, want %v", p.Status, PaymentStatusPaid)
	}
}

func TestProcessPaymentWebhook_InvalidSignatureReturnsBadRequest(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()

	gateway := &mockPaymentGateway{
		err: paymentgateway.ErrInvalidSignature,
	}

	uc := NewWebhookProcessor(
		&mockPaymentRepo{}, &mockPaymentEventRepo{},
		newMockWebhookEventRepo(),
		&mockOrderPaymentManager{},
		&mockInventoryRepo{},
		gateway,
		&mockAuditLogger{},
		&mockTransactor{}, &mockExecutor{},
	)

	err := uc.ProcessWebhook(ctx, map[string]any{
		"order_id":           orderID.String(),
		"transaction_status": "settlement",
	})
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperror.ErrTypeBadRequest {
		t.Errorf("expected BadRequest for invalid signature, got %v", err)
	}
}
