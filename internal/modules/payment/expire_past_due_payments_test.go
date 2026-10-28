package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/inventory"

	"github.com/google/uuid"
)

type mockPaymentExpiryRepo struct {
	pastDuePayments []Payment
	updatedStatuses map[uuid.UUID]PaymentStatus
	listErr         error
}

func (m *mockPaymentExpiryRepo) GetByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) (*Payment, error) {
	return nil, nil
}

func (m *mockPaymentExpiryRepo) GetByOrderID(_ context.Context, _ transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	for i := range m.pastDuePayments {
		if m.pastDuePayments[i].OrderID == orderID {
			return &m.pastDuePayments[i], nil
		}
	}
	return nil, nil
}

func (m *mockPaymentExpiryRepo) GetByOrderIDForUpdate(ctx context.Context, exec transaction.Executor, orderID uuid.UUID) (*Payment, error) {
	return m.GetByOrderID(ctx, exec, orderID)
}

func (m *mockPaymentExpiryRepo) ListByOrderIDs(_ context.Context, _ transaction.Executor, _ []uuid.UUID) ([]Payment, error) {
	return nil, nil
}

func (m *mockPaymentExpiryRepo) UpdateStatus(_ context.Context, _ transaction.Executor, id uuid.UUID, status PaymentStatus) error {
	if m.updatedStatuses == nil {
		m.updatedStatuses = make(map[uuid.UUID]PaymentStatus)
	}
	m.updatedStatuses[id] = status
	for i := range m.pastDuePayments {
		if m.pastDuePayments[i].ID == id {
			m.pastDuePayments[i].Status = status
		}
	}
	return nil
}

func (m *mockPaymentExpiryRepo) Save(_ context.Context, _ transaction.Executor, _ Payment) error {
	return nil
}

func (m *mockPaymentExpiryRepo) ListPendingGateway(_ context.Context, _ transaction.Executor, _ time.Time) ([]Payment, error) {
	return nil, nil
}

func (m *mockPaymentExpiryRepo) ListPastDuePending(_ context.Context, _ transaction.Executor, _ time.Time, _ int) ([]Payment, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.pastDuePayments, nil
}

func TestExpirePastDuePayments_NoPayments(t *testing.T) {
	paymentRepo := &mockPaymentExpiryRepo{}
	logger := &mockLogger{}
	gw := &mockPaymentGateway{}
	transactor := &mockTransactor{}

	svc := NewService(
		paymentRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		gw,
		nil,
		nil,
		transactor,
		logger,
		24*time.Hour,
		100,
		5,
	)

	svc.ExpirePastDuePayments(context.Background())
	if len(paymentRepo.updatedStatuses) != 0 {
		t.Errorf("expected 0 updated statuses, got %d", len(paymentRepo.updatedStatuses))
	}
}

func TestExpirePastDuePayments_Success(t *testing.T) {
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	gatewayOrderID := "ORDER-123"

	pastTime := time.Now().Add(-10 * time.Minute)
	pastDuePayment := Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Provider:        "gateway",
		ProviderOrderID: &gatewayOrderID,
		Status:          PaymentStatusPending,
		ExpiresAt:       &pastTime,
	}

	paymentRepo := &mockPaymentExpiryRepo{
		pastDuePayments: []Payment{pastDuePayment},
	}
	order := &mockOrder{
		ID:     orderID,
		Status: "pending",
	}
	items := []OrderItemInfo{
		{
			ProductID: productID,
			ShopID:    shopID,
			Quantity:  2,
		},
	}
	orderMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	inventoryRepo := &mockInventoryRepo{}
	gw := &mockPaymentGateway{}
	transactor := &mockTransactor{}
	logger := &mockLogger{}

	svc := NewService(
		paymentRepo,
		nil,
		nil,
		nil,
		orderMgr,
		inventoryRepo,
		gw,
		nil,
		nil,
		transactor,
		logger,
		24*time.Hour,
		100,
		5,
	)

	svc.ExpirePastDuePayments(context.Background())

	if paymentRepo.updatedStatuses[paymentID] != PaymentStatusExpired {
		t.Errorf("expected payment status expired, got %s", paymentRepo.updatedStatuses[paymentID])
	}
	if order.Status != "expired" {
		t.Errorf("expected order status expired, got %s", order.Status)
	}
	if len(inventoryRepo.releases) != 1 {
		t.Errorf("expected 1 inventory release call, got %d", len(inventoryRepo.releases))
	}
}

func TestExpirePastDuePayments_ListError(t *testing.T) {
	paymentRepo := &mockPaymentExpiryRepo{
		listErr: errors.New("db error"),
	}
	logger := &mockLogger{}
	gw := &mockPaymentGateway{}
	transactor := &mockTransactor{}

	svc := NewService(
		paymentRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		gw,
		nil,
		nil,
		transactor,
		logger,
		24*time.Hour,
		100,
		5,
	)

	svc.ExpirePastDuePayments(context.Background())
	if len(logger.errors) == 0 {
		t.Error("expected error to be logged")
	}
}

func TestExpirePastDuePayments_InventoryAnomalyLogsWarnAndFinalizesState(t *testing.T) {
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	pastTime := time.Now().Add(-10 * time.Minute)
	pastDuePayment := Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Provider:  "manual",
		Status:    PaymentStatusPending,
		ExpiresAt: &pastTime,
	}

	paymentRepo := &mockPaymentExpiryRepo{
		pastDuePayments: []Payment{pastDuePayment},
	}
	order := &mockOrder{
		ID:     orderID,
		Status: "pending",
	}
	items := []OrderItemInfo{
		{
			ProductID: productID,
			ShopID:    shopID,
			Quantity:  5,
		},
	}
	orderMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	inventoryRepo := &mockInventoryRepo{
		releaseErr: inventory.ErrInsufficientReserved,
	}
	gw := &mockPaymentGateway{}
	transactor := &mockTransactor{}
	logger := &mockLogger{}

	svc := NewService(
		paymentRepo,
		nil,
		nil,
		nil,
		orderMgr,
		inventoryRepo,
		gw,
		nil,
		nil,
		transactor,
		logger,
		24*time.Hour,
		100,
		5,
	)

	svc.ExpirePastDuePayments(context.Background())

	if paymentRepo.updatedStatuses[paymentID] != PaymentStatusExpired {
		t.Errorf("expected payment status expired despite inventory anomaly, got %s", paymentRepo.updatedStatuses[paymentID])
	}
	if order.Status != "expired" {
		t.Errorf("expected order status expired despite inventory anomaly, got %s", order.Status)
	}

	if len(logger.warns) == 0 {
		t.Error("expected WARN log for inventory anomaly, got 0")
	}
}

func TestExpirePastDuePayments_ConcurrentSettlement_AbortsExpiry(t *testing.T) {
	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	gatewayOrderID := "gw-concurrent-1"

	listedPayment := Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Provider:        "midtrans",
		ProviderOrderID: &gatewayOrderID,
		Status:          PaymentStatusPending,
		Amount:          100000,
		CreatedAt:       time.Now().Add(-25 * time.Hour),
		ExpiresAt:       pointer(time.Now().Add(-1 * time.Hour)),
	}

	paymentRepo := &mockPaymentExpiryRepo{
		pastDuePayments: []Payment{listedPayment},
	}

	order := &mockOrder{
		ID:     orderID,
		Status: "confirmed",
	}
	items := []OrderItemInfo{
		{ProductID: productID, ShopID: shopID, Quantity: 1},
	}
	orderMgr := &mockOrderPaymentManager{
		orders: map[uuid.UUID]*mockOrder{orderID: order},
		items:  map[uuid.UUID][]OrderItemInfo{orderID: items},
	}
	inventoryRepo := &mockInventoryRepo{}
	gw := &mockPaymentGateway{}
	transactor := &mockTransactor{}
	logger := &mockLogger{}

	svc := NewService(
		paymentRepo,
		nil,
		nil,
		nil,
		orderMgr,
		inventoryRepo,
		gw,
		nil,
		nil,
		transactor,
		logger,
		24*time.Hour,
		100,
		5,
	)

	// Simulate concurrent settlement happening right before lock
	paymentRepo.pastDuePayments[0].Status = PaymentStatusPaid

	svc.ExpirePastDuePayments(context.Background())

	if status, exists := paymentRepo.updatedStatuses[paymentID]; exists && status == PaymentStatusExpired {
		t.Errorf("expected payment not to be expired after concurrent settlement, got %s", status)
	}
	if order.Status != "confirmed" {
		t.Errorf("expected order status to remain confirmed, got %s", order.Status)
	}
	if len(inventoryRepo.releases) != 0 {
		t.Errorf("expected 0 inventory release calls, got %d", len(inventoryRepo.releases))
	}
}
