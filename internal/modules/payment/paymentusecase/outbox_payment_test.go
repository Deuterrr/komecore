package paymentusecase

import (
	"context"
	"testing"
	"time"

	"komecore/internal/infra/outbox"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"

	"github.com/google/uuid"
)

type mockPaymentOutboxRecorder struct {
	enqueuedEvents   []string
	enqueuedPayloads []any
	err              error
}

func (m *mockPaymentOutboxRecorder) Enqueue(_ context.Context, _ transaction.Executor, eventType string, payload any) error {
	if m.err != nil {
		return m.err
	}
	m.enqueuedEvents = append(m.enqueuedEvents, eventType)
	m.enqueuedPayloads = append(m.enqueuedPayloads, payload)
	return nil
}

func TestProcessPaymentWebhook_OutboxSettlement(t *testing.T) {
	ctx := context.Background()

	orderID := uuid.New()
	paymentID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()

	payment := &paymentdomain.Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Status:    paymentdomain.PaymentStatusPending,
		Amount:    100000,
		Provider:  "midtrans",
		CreatedAt: time.Now(),
	}
	order := &mockOrder{
		ID:            orderID,
		Status:        "pending",
		Number:        "ORD-PAY-999",
		CustomerEmail: "buyer@example.com",
		CustomerName:  "Test Buyer",
	}
	items := []OrderItemInfo{{ProductID: productID, ShopID: shopID, Quantity: 2}}

	pRepo := &mockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{paymentID: payment}}
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

	recorder := &mockPaymentOutboxRecorder{}
	uc := newWebhookUsecase(pRepo, nil, peRepo, oMgr, iRepo, gateway, nil).WithOutboxRecorder(recorder)

	err := uc.Execute(ctx, ProcessPaymentWebhookInput{
		Payload: map[string]any{
			"order_id":           orderID.String(),
			"transaction_status": "settlement",
			"payment_type":       "bank_transfer",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(recorder.enqueuedEvents) != 1 {
		t.Fatalf("expected 1 enqueued outbox event, got %d", len(recorder.enqueuedEvents))
	}
	if recorder.enqueuedEvents[0] != outbox.EventPaymentSettled {
		t.Errorf("expected %s, got %s", outbox.EventPaymentSettled, recorder.enqueuedEvents[0])
	}

	payload, ok := recorder.enqueuedPayloads[0].(outbox.PaymentSettledPayload)
	if !ok {
		t.Fatalf("expected outbox.PaymentSettledPayload, got %T", recorder.enqueuedPayloads[0])
	}
	if payload.OrderID != orderID {
		t.Errorf("expected order ID %s, got %s", orderID, payload.OrderID)
	}
	if payload.CustomerEmail != "buyer@example.com" {
		t.Errorf("expected customer email buyer@example.com, got %s", payload.CustomerEmail)
	}
	if payload.Amount != 100000 {
		t.Errorf("expected amount 100000, got %d", payload.Amount)
	}
	if payload.OrderNumber != "ORD-PAY-999" {
		t.Errorf("expected order number ORD-PAY-999, got %s", payload.OrderNumber)
	}
}
