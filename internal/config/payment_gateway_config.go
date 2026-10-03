package config

import "strings"

type PaymentGatewayConfig struct {
	Provider string // "midtrans" | "noop"
}

func LoadPaymentGatewayConfig() PaymentGatewayConfig {
	provider := strings.ToLower(strings.TrimSpace(GetEnv("PAYMENT_GATEWAY_PROVIDER", "midtrans")))
	return PaymentGatewayConfig{
		Provider: provider,
	}
}
