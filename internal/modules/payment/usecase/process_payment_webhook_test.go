package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	apperrors "komecore/internal/common/errors"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	inventoryDomain "komecore/internal/modules/inventory/domain"
	orderDomain "komecore/internal/modules/order/domain"
	orderRepo "komecore/internal/modules/order/repository"
	paymentDomain "komecore/internal/modules/payment/domain"
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
	payments  map[uuid.UUID]*paymentDomain.Payment
	updateErr error
}

func (m *mockPaymentRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*paymentDomain.Payment, error) {
	return m.payments[id], nil
}
func (m *mockPaymentRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*paymentDomain.Payment, error) {
	for _, p := range m.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, nil
}
func (m *mockPaymentRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*paymentDomain.Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}
func (m *mockPaymentRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, orderIDs []uuid.UUID) ([]paymentDomain.Payment, error) {
	var result []paymentDomain.Payment
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
func (m *mockPaymentRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status paymentDomain.PaymentStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if p, ok := m.payments[id]; ok {
		p.Status = status
		return nil
	}
	return errors.New("not found")
}
func (m *mockPaymentRepo) Save(_ context.Context, _ transaction.Executor, payment paymentDomain.Payment) error {
	m.payments[payment.ID] = &payment
	return nil
}
func (m *mockPaymentRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, since time.Time) ([]paymentDomain.Payment, error) {
	var result []paymentDomain.Payment
	for _, p := range m.payments {
		if p.Status == paymentDomain.PaymentStatusPending &&
			p.Provider == "gateway" &&
			p.ProviderOrderID != nil &&
			(p.CreatedAt.After(since) || p.CreatedAt.Equal(since)) {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (m *mockPaymentRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]paymentDomain.Payment, error) {
	return nil, nil
}

// mockPaymentAccountRepo tracks increments, decrements and supports injecting errors.
type mockPaymentAccountRepo struct{}

// mockPaymentEventRepo tracks created events and supports injecting createErr.
type mockPaymentEventRepo struct {
	events    []paymentDomain.PaymentEvent
	createErr error
}

func (m *mockPaymentEventRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*paymentDomain.PaymentEvent, error) {
	return nil, nil
}
func (m *mockPaymentEventRepo) ListByPaymentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]paymentDomain.PaymentEvent, error) {
	return nil, nil
}
func (m *mockPaymentEventRepo) Create(_ context.Context, _ transaction.Executor, event paymentDomain.PaymentEvent) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.events = append(m.events, event)
	return nil
}

// mockOrderRepo supports configurable updateErr.
type mockOrderRepo struct {
	orders    map[uuid.UUID]*orderDomain.Order
	updateErr error
}

func (m *mockOrderRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*orderDomain.Order, error) {
	return m.orders[id], nil
}
func (m *mockOrderRepo) GetByNumber(_ context.Context, _ transaction.Executor, number string) (*orderDomain.Order, error) {
	for _, o := range m.orders {
		if o.Number == number {
			return o, nil
		}
	}
	return nil, nil
}
func (m *mockOrderRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderDomain.OrderStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if o, ok := m.orders[id]; ok {
		o.Status = status
		return nil
	}
	return errors.New("not found")
}
func (m *mockOrderRepo) UpdateStatusWithSLA(_ context.Context, _ transaction.Executor, id uuid.UUID, status orderDomain.OrderStatus, confirmedAt *time.Time, expiresAt *time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if o, ok := m.orders[id]; ok {
		o.Status = status
		if confirmedAt != nil {
			o.ConfirmedAt = confirmedAt
		}
		if expiresAt != nil {
			o.HandlingExpiresAt = expiresAt
		}
		return nil
	}
	return errors.New("not found")
}
func (m *mockOrderRepo) Save(_ context.Context, _ transaction.Executor, order orderDomain.Order) error {
	m.orders[order.ID] = &order
	return nil
}
func (m *mockOrderRepo) FindOrders(_ context.Context, _ transaction.Executor, _ orderRepo.FindOrderParams) ([]orderDomain.Order, int, error) {
	return nil, 0, nil
}
func (m *mockOrderRepo) SetConfirmedAndExpiry(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ time.Time, _ time.Time) error {
	return nil
}
func (m *mockOrderRepo) FindExpiredUnfulfilledOrders(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]orderDomain.Order, error) {
	return nil, nil
}

// mockOrderItemRepo supports configurable saveBulkErr.
type mockOrderItemRepo struct {
	items       map[uuid.UUID][]orderDomain.OrderItem
	saveBulkErr error
}

func (m *mockOrderItemRepo) ListByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) ([]orderDomain.OrderItem, error) {
	return m.items[orderID], nil
}
func (m *mockOrderItemRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]orderDomain.OrderItem, error) {
	return nil, nil
}
func (m *mockOrderItemRepo) ListByShipmentID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]orderDomain.OrderItem, error) {
	return nil, nil
}
func (m *mockOrderItemRepo) SaveBulk(_ context.Context, _ transaction.Executor, _ []orderDomain.OrderItem) error {
	return m.saveBulkErr
}
func (m *mockOrderItemRepo) AssignShipment(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ []uuid.UUID) error {
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

func (m *mockInventoryRepo) GetByProductIDAndShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID, _ uuid.UUID) (*inventoryDomain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByProductID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventoryDomain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByProductIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) (map[uuid.UUID][]inventoryDomain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) ListByShopID(_ context.Context, _ transaction.Executor, _ uuid.UUID) ([]inventoryDomain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) Create(_ context.Context, _ transaction.Executor, _ *inventoryDomain.Inventory) error {
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
func (m *mockInventoryRepo) Update(_ context.Context, _ transaction.Executor, _ *inventoryDomain.Inventory) error {
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
	events      map[string]*paymentDomain.PaymentWebhookEvent // key: order_id+":"+tx_status
	upsertErr   error
	markProcErr error
	markFailErr error
	processed   []uuid.UUID
	failed      []uuid.UUID
}

func newMockWebhookEventRepo() *mockPaymentWebhookEventRepo {
	return &mockPaymentWebhookEventRepo{
		events: make(map[string]*paymentDomain.PaymentWebhookEvent),
	}
}

func (m *mockPaymentWebhookEventRepo) Upsert(
	_ context.Context,
	_ transaction.Executor,
	event paymentDomain.PaymentWebhookEvent,
) (*paymentDomain.PaymentWebhookEvent, error) {
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
// Helper: assemble a webhook usecase with sensible defaults.
// ---------------------------------------------------------------------------

func newWebhookUsecase(
	pRepo *mockPaymentRepo,
	paRepo *mockPaymentAccountRepo,
	peRepo *mockPaymentEventRepo,
	oRepo *mockOrderRepo,
	oiRepo *mockOrderItemRepo,
	iRepo *mockInventoryRepo,
	gateway paymentgateway.Provider,
	transactor transaction.Transactor,
) *ProcessPaymentWebhookUsecase {
	if transactor == nil {
		transactor = &mockTransactor{}
	}
	return NewProcessPaymentWebhookUsecase(
		pRepo, peRepo,
		newMockWebhookEventRepo(),
		oRepo, oiRepo, iRepo,
		gateway,
		&mockAuditLogger{},
		transactor, &mockExecutor{},
	)
}

// ---------------------------------------------------------------------------
// All terminal states
// ---------------------------------------------------------------------------

func TestProcessPaymentWebhook_Settlement(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	payment := &paymentDomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Status:    paymentDomain.PaymentStatusPending,
		Amount:    100000,
		Provider:  "midtrans",
		CreatedAt: time.Now(),
	}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 2}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	paRepo := &mockPaymentAccountRepo{}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
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

	err := newWebhookUsecase(pRepo, paRepo, peRepo, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusPaid {
		t.Errorf("expected payment paid, got %v", payment.Status)
	}
	if order.Status != orderDomain.OrderStatusConfirmed {
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

	payment := &paymentDomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Status:    paymentDomain.PaymentStatusPending,
		Amount:    100000,
		Provider:  "midtrans",
		CreatedAt: time.Now(),
	}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 2}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	paRepo := &mockPaymentAccountRepo{}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusExpire,
			RawStatus:      "expire",
			GrossAmount:    100000,
		},
	}

	err := newWebhookUsecase(pRepo, paRepo, peRepo, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "expire"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusExpired {
		t.Errorf("expected payment expired, got %v", payment.Status)
	}
	if order.Status != orderDomain.OrderStatusExpired {
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
			RawStatus:      "cancel",
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusCancelled {
		t.Errorf("expected payment cancelled, got %v", payment.Status)
	}
	if order.Status != orderDomain.OrderStatusCancelled {
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 3}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusDeny,
			RawStatus:      "deny",
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "deny"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusFailed {
		t.Errorf("expected payment failed, got %v", payment.Status)
	}
	if order.Status != orderDomain.OrderStatusCancelled {
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 50000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusPending,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, peRepo, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "pending"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusPending {
		t.Errorf("payment status changed unexpectedly for pending: %v", payment.Status)
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 50000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusChallenge,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, peRepo, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "challenge"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payment.Status != paymentDomain.PaymentStatusPending {
		t.Errorf("payment status changed unexpectedly for challenge: %v", payment.Status)
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPaid, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusConfirmed}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, peRepo, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusCancelled, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusCancelled}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	iRepo := &mockInventoryRepo{}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, peRepo, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"}})
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

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{}},
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}},
		&mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})

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

	err := newWebhookUsecase(&mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{}},
		&mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{}},
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}},
		&mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": "not-a-uuid", "transaction_status": "settlement"}})

	if err == nil {
		t.Fatal("expected error for invalid order ID in gateway response")
	}
}

func TestProcessPaymentWebhook_GatewayParseError(t *testing.T) {
	ctx := context.Background()
	gateway := &mockPaymentGateway{err: errors.New("midtrans: gateway unreachable")}

	err := newWebhookUsecase(&mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{}},
		&mockPaymentAccountRepo{}, &mockPaymentEventRepo{},
		&mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{}},
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}},
		&mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": uuid.New().String(), "transaction_status": "settlement"}})

	if err == nil {
		t.Fatal("expected error when gateway.ParseNotification fails")
	}
}

func TestProcessPaymentWebhook_PaymentRepoUpdateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	pRepo := &mockPaymentRepo{
		payments:  map[uuid.UUID]*paymentDomain.Payment{paymentID: payment},
		updateErr: errors.New("db: update failed"),
	}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, &mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})

	if err == nil {
		t.Fatal("expected error when payment repo update fails")
	}
}

func TestProcessPaymentWebhook_OrderRepoUpdateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{
		orders:    map[uuid.UUID]*orderDomain.Order{orderID: order},
		updateErr: errors.New("db: order update failed"),
	}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, &mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})

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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
	iRepo := &mockInventoryRepo{commitErr: errors.New("inventory: stock underflow")}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})
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

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	items := []orderDomain.OrderItem{{ProductID: productID, ShopID: shopID, Quantity: 1}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	oiRepo := &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{orderID: items}}
	iRepo := &mockInventoryRepo{releaseErr: errors.New("inventory: release error")}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusExpire,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, oiRepo, iRepo, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "expire"}})
	if err == nil {
		t.Fatal("expected error when inventory release fails")
	}
}

func TestProcessPaymentWebhook_PaymentEventCreateFails(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusPending}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	peRepo := &mockPaymentEventRepo{createErr: errors.New("db: event insert failed")}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusSettlement,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, peRepo, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, &mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "settlement"}})
	if err == nil {
		t.Fatal("expected error when payment event creation fails")
	}
}

func TestProcessPaymentWebhook_InvalidOrderStatusTransition(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	payment := &paymentDomain.Payment{ID: paymentID, OrderID: orderID, Status: paymentDomain.PaymentStatusPending, Amount: 100000, Provider: "midtrans", CreatedAt: time.Now()}
	// Order is already delivered, so transition to cancelled is invalid
	order := &orderDomain.Order{ID: orderID, Status: orderDomain.OrderStatusDelivered}
	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID: orderID.String(),
			Status:         paymentgateway.NotificationStatusCancel,
		},
	}

	err := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo,
		&mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}}, &mockInventoryRepo{}, gateway, nil).
		Execute(ctx, ProcessPaymentWebhookInput{Payload: map[string]any{"order_id": orderID.String(), "transaction_status": "cancel"}})
	if err == nil {
		t.Fatal("expected error for invalid order status transition via webhook, got nil")
	}
}

func TestProcessPaymentWebhook_DirectNotificationResult_BypassesParseNotification(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()
	paymentID := uuid.New()

	pRepo := &mockPaymentRepo{
		payments: map[uuid.UUID]*paymentDomain.Payment{
			paymentID: {
				ID:      paymentID,
				OrderID: orderID,
				Status:  paymentDomain.PaymentStatusPending,
				Amount:  100000,
			},
		},
	}
	peRepo := &mockPaymentEventRepo{}
	oRepo := &mockOrderRepo{
		orders: map[uuid.UUID]*orderDomain.Order{
			orderID: {
				ID:     orderID,
				Status: orderDomain.OrderStatusPending,
				Total:  100000,
			},
		},
	}
	invRepo := &mockInventoryRepo{}
	// gateway with error if ParseNotification is called
	gateway := &mockPaymentGateway{
		err: errors.New("should not be called"),
	}

	uc := NewProcessPaymentWebhookUsecase(
		pRepo, peRepo,
		newMockWebhookEventRepo(),
		oRepo, &mockOrderItemRepo{items: map[uuid.UUID][]orderDomain.OrderItem{}},
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
	if p.Status != paymentDomain.PaymentStatusPaid {
		t.Errorf("payment status = %v, want %v", p.Status, paymentDomain.PaymentStatusPaid)
	}
}

func TestProcessPaymentWebhook_InvalidSignatureReturnsBadRequest(t *testing.T) {
	ctx := context.Background()
	orderID := uuid.New()

	gateway := &mockPaymentGateway{
		err: paymentgateway.ErrInvalidSignature,
	}

	uc := NewProcessPaymentWebhookUsecase(
		&mockPaymentRepo{}, &mockPaymentEventRepo{},
		newMockWebhookEventRepo(),
		&mockOrderRepo{}, &mockOrderItemRepo{},
		&mockInventoryRepo{},
		gateway,
		&mockAuditLogger{},
		&mockTransactor{}, &mockExecutor{},
	)

	err := uc.Execute(ctx, ProcessPaymentWebhookInput{
		Payload: map[string]any{
			"order_id":           orderID.String(),
			"transaction_status": "settlement",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != apperrors.ErrTypeBadRequest {
		t.Errorf("expected BadRequest for invalid signature, got %v", err)
	}
}
