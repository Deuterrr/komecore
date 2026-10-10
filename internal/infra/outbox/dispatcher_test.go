package outbox

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"komecore/pkg/mailer"

	"github.com/google/uuid"
)

func TestEmailDispatcher_OrderCreated(t *testing.T) {
	mockMailer := mailer.NewMockSender()
	dispatcher := NewEmailDispatcher(mockMailer)

	payload := OrderCreatedPayload{
		OrderID:       uuid.New(),
		OrderNumber:   "ORD-12345",
		CustomerEmail: "customer@example.com",
		CustomerName:  "Jane Doe",
		Total:         150000,
		Items: []OrderCreatedItem{
			{
				ProductName: "Specialty Coffee Beans",
				Quantity:    2,
				UnitPrice:   75000,
				Subtotal:    150000,
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	event := Event{
		ID:        uuid.New(),
		EventType: EventOrderCreated,
		Payload:   payloadBytes,
	}

	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}

	if len(mockMailer.SentMails) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(mockMailer.SentMails))
	}

	sent := mockMailer.SentMails[0]
	if sent.To != "customer@example.com" {
		t.Errorf("expected To: customer@example.com, got %s", sent.To)
	}
	if !strings.Contains(sent.Subject, "ORD-12345") {
		t.Errorf("expected Subject to contain order number, got %s", sent.Subject)
	}
	if !strings.Contains(sent.Text, "Specialty Coffee Beans") {
		t.Errorf("expected text body to contain product name")
	}
	if sent.HTML == nil || !strings.Contains(*sent.HTML, "150000") {
		t.Errorf("expected HTML body to contain total amount")
	}
}

func TestEmailDispatcher_PaymentSettled(t *testing.T) {
	mockMailer := mailer.NewMockSender()
	dispatcher := NewEmailDispatcher(mockMailer)

	payload := PaymentSettledPayload{
		OrderID:       uuid.New(),
		OrderNumber:   "ORD-67890",
		CustomerEmail: "buyer@example.com",
		CustomerName:  "John Smith",
		Amount:        200000,
		PaymentMethod: "BCA Virtual Account",
		PaidAt:        time.Now().UTC(),
	}
	payloadBytes, _ := json.Marshal(payload)

	event := Event{
		ID:        uuid.New(),
		EventType: EventPaymentSettled,
		Payload:   payloadBytes,
	}

	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}

	if len(mockMailer.SentMails) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(mockMailer.SentMails))
	}

	sent := mockMailer.SentMails[0]
	if sent.To != "buyer@example.com" {
		t.Errorf("expected To: buyer@example.com, got %s", sent.To)
	}
	if !strings.Contains(sent.Subject, "Payment Receipt") {
		t.Errorf("expected Subject to contain Payment Receipt, got %s", sent.Subject)
	}
	if !strings.Contains(sent.Text, "BCA Virtual Account") {
		t.Errorf("expected text to mention payment method")
	}
}

func TestEmailDispatcher_ShipmentDispatched(t *testing.T) {
	mockMailer := mailer.NewMockSender()
	dispatcher := NewEmailDispatcher(mockMailer)

	payload := ShipmentDispatchedPayload{
		OrderID:        uuid.New(),
		OrderNumber:    "ORD-11111",
		CustomerEmail:  "buyer@example.com",
		CustomerName:   "Alice",
		TrackingNumber: "JNE-TRACK-999",
		Courier:        "jne",
		Service:        "REG",
	}
	payloadBytes, _ := json.Marshal(payload)

	event := Event{
		ID:        uuid.New(),
		EventType: EventShipmentDispatched,
		Payload:   payloadBytes,
	}

	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}

	if len(mockMailer.SentMails) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(mockMailer.SentMails))
	}

	sent := mockMailer.SentMails[0]
	if !strings.Contains(sent.Text, "JNE-TRACK-999") {
		t.Errorf("expected tracking number in text body")
	}
}

func TestEmailDispatcher_UnhandledEventIgnored(t *testing.T) {
	mockMailer := mailer.NewMockSender()
	dispatcher := NewEmailDispatcher(mockMailer)

	event := Event{
		ID:        uuid.New(),
		EventType: "custom.unknown.event",
		Payload:   json.RawMessage(`{}`),
	}

	err := dispatcher.Dispatch(context.Background(), event)
	if err != nil {
		t.Errorf("expected no error for unhandled event, got %v", err)
	}
	if len(mockMailer.SentMails) != 0 {
		t.Errorf("expected 0 emails sent, got %d", len(mockMailer.SentMails))
	}
}
