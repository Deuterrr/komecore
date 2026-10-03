package config

import (
	"strconv"
)

type MidTransConfig struct {
	IsProduction bool
	URL          string
	MerchantID   string
	ClientKey    string
	ServerKey    string
}

func LoadMidTransConfig() MidTransConfig {
	isProduction, _ := strconv.ParseBool(GetEnv("MIDTRANS_IS_PRODUCTION", "false"))

	return MidTransConfig{
		IsProduction: isProduction,
		URL:          GetEnv("MIDTRANS_URL", ""),
		MerchantID:   GetEnv("MIDTRANS_MERCHANT_ID", ""),
		ClientKey:    GetEnv("MIDTRANS_CLIENT_KEY", ""),
		ServerKey:    GetEnv("MIDTRANS_SERVER_KEY", ""),
	}
}
