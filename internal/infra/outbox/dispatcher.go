package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"komecore/pkg/mailer"
)

// Dispatcher defines the event dispatching contract.
type Dispatcher interface {
	Dispatch(ctx context.Context, event Event) error
}

type EmailDispatcher struct {
	mailer mailer.Sender
}

func NewEmailDispatcher(m mailer.Sender) *EmailDispatcher {
	return &EmailDispatcher{mailer: m}
}

func (d *EmailDispatcher) Dispatch(ctx context.Context, event Event) error {
	switch event.EventType {
	case EventOrderCreated:
		return d.handleOrderCreated(ctx, event)
	case EventPaymentSettled:
		return d.handlePaymentSettled(ctx, event)
	case EventShipmentDispatched:
		return d.handleShipmentDispatched(ctx, event)
	default:
		// Unsupported or unhandled event types do not block the worker
		return nil
	}
}

func (d *EmailDispatcher) handleOrderCreated(_ context.Context, event Event) error {
	var payload OrderCreatedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal order.created payload: %w", err)
	}

	if payload.CustomerEmail == "" {
		return nil
	}

	subject := fmt.Sprintf("Order Confirmation - #%s", payload.OrderNumber)

	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("Hello %s,\n\n", payload.CustomerName))
	textBuilder.WriteString(fmt.Sprintf("Thank you for your order #%s!\n\nOrder Details:\n", payload.OrderNumber))
	for _, item := range payload.Items {
		textBuilder.WriteString(fmt.Sprintf("- %s x%d — IDR %d (Subtotal: IDR %d)\n",
			item.ProductName, item.Quantity, item.UnitPrice, item.Subtotal))
	}
	textBuilder.WriteString(fmt.Sprintf("\nGrand Total: IDR %d\n", payload.Total))
	textBuilder.WriteString("\nWe will notify you once payment is received and your items are being prepared.\n\nBest regards,\nKomeCore Team")

	textBody := textBuilder.String()
	htmlBody := fmt.Sprintf(`
		<h2>Order Confirmation</h2>
		<p>Hello <strong>%s</strong>,</p>
		<p>Thank you for your order <strong>#%s</strong>!</p>
		<h3>Order Breakdown</h3>
		<ul>
	`, payload.CustomerName, payload.OrderNumber)

	for _, item := range payload.Items {
		htmlBody += fmt.Sprintf("<li>%s x%d &mdash; IDR %d (Subtotal: IDR %d)</li>",
			item.ProductName, item.Quantity, item.UnitPrice, item.Subtotal)
	}

	htmlBody += fmt.Sprintf(`
		</ul>
		<p><strong>Grand Total: IDR %d</strong></p>
		<p>We will notify you once payment is confirmed.</p>
		<p>Best regards,<br/>KomeCore Team</p>
	`, payload.Total)

	return d.mailer.Send(mailer.SendInput{
		To:      payload.CustomerEmail,
		Subject: subject,
		Text:    textBody,
		HTML:    &htmlBody,
	})
}

func (d *EmailDispatcher) handlePaymentSettled(_ context.Context, event Event) error {
	var payload PaymentSettledPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment.settled payload: %w", err)
	}

	if payload.CustomerEmail == "" {
		return nil
	}

	subject := fmt.Sprintf("Payment Receipt - #%s", payload.OrderNumber)

	textBody := fmt.Sprintf(
		"Hello %s,\n\nWe have received your payment of IDR %d for order #%s via %s.\nPaid At: %s\n\nYour order is now being processed by the merchant.\n\nBest regards,\nKomeCore Team",
		payload.CustomerName,
		payload.Amount,
		payload.OrderNumber,
		payload.PaymentMethod,
		payload.PaidAt.Format("2006-01-02 15:04:05 UTC"),
	)

	htmlBody := fmt.Sprintf(`
		<h2>Payment Receipt</h2>
		<p>Hello <strong>%s</strong>,</p>
		<p>We have successfully received your payment for order <strong>#%s</strong>.</p>
		<ul>
			<li><strong>Amount Paid:</strong> IDR %d</li>
			<li><strong>Payment Method:</strong> %s</li>
			<li><strong>Timestamp:</strong> %s</li>
		</ul>
		<p>Your order is now being processed and readied for shipment.</p>
		<p>Best regards,<br/>KomeCore Team</p>
	`, payload.CustomerName, payload.OrderNumber, payload.Amount, payload.PaymentMethod, payload.PaidAt.Format("2006-01-02 15:04:05 UTC"))

	return d.mailer.Send(mailer.SendInput{
		To:      payload.CustomerEmail,
		Subject: subject,
		Text:    textBody,
		HTML:    &htmlBody,
	})
}

func (d *EmailDispatcher) handleShipmentDispatched(_ context.Context, event Event) error {
	var payload ShipmentDispatchedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal shipment.dispatched payload: %w", err)
	}

	if payload.CustomerEmail == "" {
		return nil
	}

	subject := fmt.Sprintf("Shipment Dispatched - #%s", payload.OrderNumber)

	textBody := fmt.Sprintf(
		"Hello %s,\n\nGreat news! Your package for order #%s has been dispatched via %s (%s).\nTracking Number: %s\n\nTrack your order in real-time in your account dashboard.\n\nBest regards,\nKomeCore Team",
		payload.CustomerName,
		payload.OrderNumber,
		payload.Courier,
		payload.Service,
		payload.TrackingNumber,
	)

	htmlBody := fmt.Sprintf(`
		<h2>Your Order Has Been Dispatched!</h2>
		<p>Hello <strong>%s</strong>,</p>
		<p>Package for order <strong>#%s</strong> is on its way.</p>
		<ul>
			<li><strong>Courier:</strong> %s</li>
			<li><strong>Service:</strong> %s</li>
			<li><strong>Tracking / Waybill:</strong> %s</li>
		</ul>
		<p>Best regards,<br/>KomeCore Team</p>
	`, payload.CustomerName, payload.OrderNumber, payload.Courier, payload.Service, payload.TrackingNumber)

	return d.mailer.Send(mailer.SendInput{
		To:      payload.CustomerEmail,
		Subject: subject,
		Text:    textBody,
		HTML:    &htmlBody,
	})
}
