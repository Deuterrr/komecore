package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentgateway "komecore/internal/infra/payment-gateway"
	orderDomain "komecore/internal/modules/order/domain"
	paymentDomain "komecore/internal/modules/payment/domain"

	"github.com/google/uuid"
)

func TestCheckPaymentStatus_Success_Paid(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &orderDomain.Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     orderDomain.OrderStatusPending,
	}

	providerOrderID := orderID.String()
	payment := &paymentDomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentDomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{
		result: &paymentgateway.NotificationResult{
			GatewayOrderID:       orderID.String(),
			GatewayTransactionID: "tx-123",
			Status:               paymentgateway.NotificationStatusSettlement,
			RawStatus:            "settlement",
			GrossAmount:          100000,
		},
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, &mockOrderItemRepo{}, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oRepo, pRepo, gateway, webhookUsecase, &mockExecutor{})

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

	if res.Status != paymentDomain.PaymentStatusPaid {
		t.Errorf("expected Status to be Paid, got %v", res.Status)
	}

	if payment.Status != paymentDomain.PaymentStatusPaid {
		t.Errorf("expected payment to be updated to Paid, got %v", payment.Status)
	}

	if order.Status != orderDomain.OrderStatusConfirmed {
		t.Errorf("expected order status to be Confirmed, got %v", order.Status)
	}
}

func TestCheckPaymentStatus_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, &mockOrderItemRepo{}, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oRepo, pRepo, gateway, webhookUsecase, &mockExecutor{})

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

	order := &orderDomain.Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     orderDomain.OrderStatusPending,
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, &mockOrderItemRepo{}, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oRepo, pRepo, gateway, webhookUsecase, &mockExecutor{})

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

	order := &orderDomain.Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     orderDomain.OrderStatusConfirmed,
	}

	providerOrderID := orderID.String()
	payment := &paymentDomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentDomain.PaymentStatusPaid,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, &mockOrderItemRepo{}, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oRepo, pRepo, gateway, webhookUsecase, &mockExecutor{})

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

	if res.Status != paymentDomain.PaymentStatusPaid {
		t.Errorf("expected Status to remain Paid, got %v", res.Status)
	}
}

func TestCheckPaymentStatus_GatewayError(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	customerID := uuid.New()
	paymentID := uuid.New()

	order := &orderDomain.Order{
		ID:         orderID,
		CustomerID: customerID,
		Status:     orderDomain.OrderStatusPending,
	}

	providerOrderID := orderID.String()
	payment := &paymentDomain.Payment{
		ID:              paymentID,
		OrderID:         orderID,
		Status:          paymentDomain.PaymentStatusPending,
		Provider:        "gateway",
		ProviderOrderID: &providerOrderID,
		Amount:          100000,
		CreatedAt:       time.Now(),
	}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentDomain.Payment{paymentID: payment}}
	oRepo := &mockOrderRepo{orders: map[uuid.UUID]*orderDomain.Order{orderID: order}}
	gateway := &mockPaymentGateway{
		err: errors.New("gateway error"),
	}

	webhookUsecase := newWebhookUsecase(pRepo, &mockPaymentAccountRepo{}, &mockPaymentEventRepo{}, oRepo, &mockOrderItemRepo{}, &mockInventoryRepo{}, gateway, nil)
	usecase := NewCheckPaymentStatusUsecase(oRepo, pRepo, gateway, webhookUsecase, &mockExecutor{})

	_, err := usecase.Execute(ctx, CheckPaymentStatusInput{
		OrderID:    orderID,
		CustomerID: customerID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
