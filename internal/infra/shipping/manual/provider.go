package manual

import (
	"context"

	shipping "komecore/internal/infra/shipping"
)

// manualShippingProvider is a null-object implementation of shipping.LogisticsProvider.
// It performs no I/O and carries no configuration. All logistics data is
// supplied directly by staff via the request body.
type manualShippingProvider struct{}

// NewManualShippingProvider returns a LogisticsProvider that does not call any
// external API. Select it by setting LOGISTICS_PROVIDER=manual.
func NewManualShippingProvider() shipping.LogisticsProvider {
	return &manualShippingProvider{}
}

// CreateOrder does not call any external API.
// It echoes ManualTrackingNumber from the input as the result so the
// caller stores whatever tracking number staff provided. If none was
// given the result is an empty string, which the domain model stores
// as a nil *string (already valid — tracking numbers are optional).
func (p *manualShippingProvider) CreateOrder(
	_ context.Context,
	input shipping.CreateOrderInput,
) (*shipping.CreateOrderResult, error) {
	tracking := ""
	if input.ManualTrackingNumber != nil {
		tracking = *input.ManualTrackingNumber
	}

	return &shipping.CreateOrderResult{
		KomerceOrderNo: "",
		TrackingNumber: tracking,
	}, nil
}

// CancelOrder is a no-op for manual shipping provider.
func (p *manualShippingProvider) CancelOrder(
	_ context.Context,
	_ string,
) error {
	return nil
}

// TrackShipment returns an empty event list.
// Manual shipments have no automated tracking feed.
func (p *manualShippingProvider) TrackShipment(
	_ context.Context,
	_ shipping.TrackShipmentInput,
) ([]shipping.TrackingEvent, error) {
	return []shipping.TrackingEvent{}, nil
}
