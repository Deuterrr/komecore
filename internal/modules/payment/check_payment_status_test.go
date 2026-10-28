package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"

	"github.com/google/uuid"
)

func TestCheckPaymentStatus_Success_Paid(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:         orderID,
		CustomerID: customerID,
		Status:     "pending",
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
	svc := NewService(pRepo, nil, nil, nil, oMgr, nil, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	res, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Synced {
		t.Errorf("expected Synced to be true")
	}

	if res.Status != PaymentStatusPaid {
		t.Errorf("expected Status to be Paid, got %v", res.Status)
	}

	if pmt.Status != PaymentStatusPaid {
		t.Errorf("expected payment to be updated to Paid, got %v", pmt.Status)
	}

	if order.Status != "confirmed" {
		t.Errorf("expected order status to be Confirmed, got %v", order.Status)
	}
}

func TestCheckPaymentStatus_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{}}
	gateway := &mockPaymentGateway{}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	svc := NewService(pRepo, nil, nil, nil, oMgr, nil, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	_, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckPaymentStatus_WrongCustomer(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	wrongCustomerID := uuid.New()

	order := &mockOrder{
		ID:         orderID,
		CustomerID: customerID,
		Status:     "pending",
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	svc := NewService(pRepo, nil, nil, nil, oMgr, nil, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	_, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: wrongCustomerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckPaymentStatus_NotPending(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:         orderID,
		CustomerID: customerID,
		Status:     "confirmed",
	}

	providerOrderID := orderID.String()
	pmt := &Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          PaymentStatusPaid,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	svc := NewService(pRepo, nil, nil, nil, oMgr, nil, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	res, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Synced {
		t.Errorf("expected Synced to be false")
	}

	if res.Status != PaymentStatusPaid {
		t.Errorf("expected Status to remain Paid, got %v", res.Status)
	}
}

func TestCheckPaymentStatus_GatewayError(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:         orderID,
		CustomerID: customerID,
		Status:     "pending",
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
		err: errors.New("gateway error"),
	}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	svc := NewService(pRepo, nil, nil, nil, oMgr, nil, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	_, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckPaymentStatus_Success_Deny(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &mockOrder{
		ID:         orderID,
		CustomerID: customerID,
		Status:     "pending",
	}

	providerOrderID := orderID.String()
	pmt := &Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          PaymentStatusPending,
		Provider:        "midtrans",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*Payment{paymentID: pmt}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID:       providerOrderID,
			GatewayTransactionID: "tx-deny-123",
			Status:               paymentgateway.NotificationStatusDeny,
			RawStatus:            "deny",
			GrossAmount:          100000,
		},
	}

	webhookProcessor := newWebhookProcessor(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	svc := NewService(pRepo, nil, nil, nil, oMgr, &mockInventoryRepo{}, gateway, webhookProcessor, &mockExecutor{}, &mockTransactor{}, &mockLogger{}, 24*time.Hour, 100, 5)

	res, err := svc.CheckPaymentStatus(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != PaymentStatusFailed {
		t.Errorf("expected status %s, got %s", PaymentStatusFailed, res.Status)
	}
	if !res.Synced {
		t.Error("expected synced to be true")
	}

	if pmt.Status != PaymentStatusFailed {
		t.Errorf("expected payment status %s, got %s", PaymentStatusFailed, pmt.Status)
	}
}
