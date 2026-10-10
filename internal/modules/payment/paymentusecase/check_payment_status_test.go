package paymentusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/infra/paymentgateway"
	"komecore/internal/modules/payment/paymentdomain"

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
	usecase := NewCheckPaymentStatusUsecase(oMgr, pRepo, gateway, webhookUsecase, &mockExecutor{})

	res, err := usecase.Execute(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Synced {
		t.Errorf("expected Synced to be true")
	}

	if res.Status != paymentdomain.PaymentStatusPaid {
		t.Errorf("expected Status to be Paid, got %v", res.Status)
	}

	if payment.Status != paymentdomain.PaymentStatusPaid {
		t.Errorf("expected payment to be updated to Paid, got %v", payment.Status)
	}

	if order.Status != "confirmed" {
		t.Errorf("expected order status to be Confirmed, got %v", order.Status)
	}
}

func TestCheckPaymentStatus_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oMgr, pRepo, gateway, webhookUsecase, &mockExecutor{})

	_, err := usecase.Execute(ctx, CheckPaymentStatusInput{
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

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oMgr, pRepo, gateway, webhookUsecase, &mockExecutor{})

	_, err := usecase.Execute(ctx, CheckPaymentStatusInput{
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
	payment := &paymentdomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentdomain.PaymentStatusPaid,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{paymentID: payment}}
	oMgr := &mockOrderPaymentManager{orders: map[uuid.UUID]*mockOrder{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oMgr, pRepo, gateway, webhookUsecase, &mockExecutor{})

	res, err := usecase.Execute(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Synced {
		t.Errorf("expected Synced to be false")
	}

	if res.Status != paymentdomain.PaymentStatusPaid {
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
		err: errors.New("gateway error"),
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oMgr, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oMgr, pRepo, gateway, webhookUsecase, &mockExecutor{})

	_, err := usecase.Execute(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
