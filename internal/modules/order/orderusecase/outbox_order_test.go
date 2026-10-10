package orderusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"komecore/internal/infra/outbox"
	paymentgateway "komecore/internal/infra/payment-gateway"
	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/order/orderdomain"
	"komecore/internal/modules/payment/paymentdomain"

	"github.com/google/uuid"
)

type mockOutboxRecorder struct {
	enqueuedEvents   []string
	enqueuedPayloads []any
	err              error
}

func (m *mockOutboxRecorder) Enqueue(_ context.Context, _ transaction.Executor, eventType string, payload any) error {
	if m.err != nil {
		return m.err
	}
	m.enqueuedEvents = append(m.enqueuedEvents, eventType)
	m.enqueuedPayloads = append(m.enqueuedPayloads, payload)
	return nil
}

func TestCreateOrder_OutboxEnqueued(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)
	providerOrderID := uuid.New().String()

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "midtrans-tx-001",
			GatewayOrderID:       providerOrderID,
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
			Instructions: []paymentgateway.PaymentInstruction{
				{Type: "qris", Label: "QRIS", Value: "qr-string-value"},
			},
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	paymentStore := &coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}}
	orderStore := &coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}}

	recorder := &mockOutboxRecorder{}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		nil,
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		paymentStore,
		orderStore,
		nil,
	).WithOutboxRecorder(recorder)

	in := coInput(customerID, methodID, false, productID, shopID)
	res, err := uc.Execute(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if len(recorder.enqueuedEvents) != 1 {
		t.Fatalf("expected 1 enqueued outbox event, got %d", len(recorder.enqueuedEvents))
	}
	if recorder.enqueuedEvents[0] != outbox.EventOrderCreated {
		t.Errorf("expected event %s, got %s", outbox.EventOrderCreated, recorder.enqueuedEvents[0])
	}

	payload, ok := recorder.enqueuedPayloads[0].(outbox.OrderCreatedPayload)
	if !ok {
		t.Fatalf("expected payload type outbox.OrderCreatedPayload, got %T", recorder.enqueuedPayloads[0])
	}
	if payload.OrderID != res.OrderID {
		t.Errorf("expected payload order ID %s, got %s", res.OrderID, payload.OrderID)
	}
	if payload.CustomerEmail != acc.Email {
		t.Errorf("expected customer email %s, got %s", acc.Email, payload.CustomerEmail)
	}
}

func TestCreateOrder_OutboxFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.New()
	productID := uuid.New()
	shopID := uuid.New()
	methodID := uuid.New()

	user := coDefaultUser()
	acc := coDefaultAccount(user.ID)
	method := coDefaultMethod(methodID, "qris")
	pricing := coDefaultPricing(productID, shopID)

	gateway := &coMockGateway{
		chargeResp: &paymentgateway.ChargeResponse{
			GatewayTransactionID: "midtrans-tx-001",
			GatewayOrderID:       uuid.New().String(),
			PaymentType:          "qris",
			GrossAmount:          115000,
			Status:               "pending",
		},
	}

	recorder := &mockOutboxRecorder{
		err: errors.New("outbox table write error"),
	}

	uc := buildUC(nil, gateway,
		&coMockPricingService{result: pricing},
		&coMockPaymentMethodRepo{method: method},
		nil,
		&coMockUserRepo{user: user},
		&coMockAccountRepo{account: acc},
		&coMockInventoryRepo{},
		&coMockPaymentRepo{payments: map[uuid.UUID]*paymentdomain.Payment{}},
		&coMockOrderRepo{orders: map[uuid.UUID]*orderdomain.Order{}},
		nil,
	).WithOutboxRecorder(recorder)

	in := coInput(customerID, methodID, false, productID, shopID)
	_, err := uc.Execute(ctx, in)
	if err == nil {
		t.Fatal("expected error when outbox enqueue fails")
	}
}
