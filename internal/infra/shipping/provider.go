package shipping

import (
	"context"
	"math"
	"strings"
	"time"
)

type CalculateRatesInput struct {
	OriginID      int
	DestinationID int
	Weight        int // grams
	Couriers      []string
	PriceFilter   *string
}

type RateOption struct {
	Name                   string
	Code                   string
	Service                string
	Description            string
	Cost                   int64
	Etd                    string
	EstimatedDurationHours *float64
	SLAConfidenceScore     *float64
	DeliveryStatus         *string
}

type ShippingProvider interface {
	CalculateShippingFee(weightKg float64, courierCode string) (int64, error)
	CalculateRates(ctx context.Context, input CalculateRatesInput) ([]RateOption, error)
}

// Provider is an alias for ShippingProvider for convenience.
type Provider = ShippingProvider

type simpleShippingProvider struct{}

func NewSimpleProvider() ShippingProvider {
	return &simpleShippingProvider{}
}

func (s *simpleShippingProvider) CalculateShippingFee(weightKg float64, courierCode string) (int64, error) {
	baseWeight := math.Ceil(weightKg)
	if baseWeight < 1 {
		baseWeight = 1
	}

	code := strings.ToUpper(strings.TrimSpace(courierCode))
	switch code {
	case "JNE":
		return 15000 * int64(baseWeight), nil
	case "JNT", "J&T":
		return 12000 * int64(baseWeight), nil
	case "SICEPAT":
		return 11000 * int64(baseWeight), nil
	case "FLAT":
		return 10000, nil
	default:
		return 10000 * int64(baseWeight), nil
	}
}

func (s *simpleShippingProvider) CalculateRates(_ context.Context, input CalculateRatesInput) ([]RateOption, error) {
	weightKg := float64(input.Weight) / 1000.0
	if weightKg <= 0 {
		weightKg = 1.0
	}

	couriers := input.Couriers
	if len(couriers) == 0 {
		couriers = []string{"jne", "jnt", "sicepat", "flat"}
	}

	knownOptions := map[string]struct {
		name        string
		code        string
		service     string
		description string
		etd         string
		hours       float64
	}{
		"JNE":     {name: "JNE Express", code: "jne", service: "REG", description: "JNE Regular Service", etd: "2-3 days", hours: 48},
		"JNT":     {name: "J&T Express", code: "jnt", service: "EZ", description: "J&T Regular Service", etd: "1-2 days", hours: 24},
		"SICEPAT": {name: "SiCepat", code: "sicepat", service: "SIUNT", description: "SiCepat Untung Service", etd: "1-2 days", hours: 24},
		"FLAT":    {name: "Flat Rate", code: "flat", service: "STD", description: "Flat Standard Delivery", etd: "3-5 days", hours: 72},
	}

	var results []RateOption
	for _, c := range couriers {
		cUpper := strings.ToUpper(strings.TrimSpace(c))
		cost, _ := s.CalculateShippingFee(weightKg, cUpper)

		opt, exists := knownOptions[cUpper]
		if !exists {
			opt = struct {
				name        string
				code        string
				service     string
				description string
				etd         string
				hours       float64
			}{
				name:        strings.ToUpper(c),
				code:        strings.ToLower(c),
				service:     "REG",
				description: strings.ToUpper(c) + " Regular Service",
				etd:         "2-4 days",
				hours:       48,
			}
		}

		durationHours := opt.hours
		confScore := 0.95
		status := "available"

		results = append(results, RateOption{
			Name:                   opt.name,
			Code:                   opt.code,
			Service:                opt.service,
			Description:            opt.description,
			Cost:                   cost,
			Etd:                    opt.etd,
			EstimatedDurationHours: &durationHours,
			SLAConfidenceScore:     &confScore,
			DeliveryStatus:         &status,
		})
	}

	if input.PriceFilter != nil && len(results) > 0 {
		switch strings.ToLower(*input.PriceFilter) {
		case "cheapest", "lowest":
			best := results[0]
			for _, r := range results[1:] {
				if r.Cost < best.Cost {
					best = r
				}
			}
			results = []RateOption{best}
		case "fastest":
			best := results[0]
			for _, r := range results[1:] {
				if r.EstimatedDurationHours != nil && best.EstimatedDurationHours != nil && *r.EstimatedDurationHours < *best.EstimatedDurationHours {
					best = r
				}
			}
			results = []RateOption{best}
		}
	}

	return results, nil
}

type CreateOrderInput struct {
	OriginAreaID      int
	DestinationAreaID int
	UniqueOrderID     string
	CourierCode       string
	CourierService    string
	Weight            int

	ItemName  string
	ItemPrice int64
	ItemQty   int

	ShipperName     string
	ShipperPhone    string
	ShipperAddress  string
	ReceiverName    string
	ReceiverPhone   string
	ReceiverAddress string

	ManualTrackingNumber *string
}

type CreateOrderResult struct {
	KomerceOrderNo string
	TrackingNumber string
}

type TrackShipmentInput struct {
	Courier        string
	TrackingNumber string
	LastPhone      *string
}

type TrackingEvent struct {
	Status      string
	Description string
	Location    string
	Timestamp   time.Time
}

type LogisticsProvider interface {
	CreateOrder(ctx context.Context, input CreateOrderInput) (*CreateOrderResult, error)
	CancelOrder(ctx context.Context, komerceOrderNo string) error
	TrackShipment(ctx context.Context, input TrackShipmentInput) ([]TrackingEvent, error)
}
