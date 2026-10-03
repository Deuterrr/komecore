package config

// LogisticsConfig controls which
// logistics provider is used at runtime.
//
// Set the LOGISTICS_PROVIDER environment
// variable to one of:
//
//	"manual" — no external API; staff supply tracking info directly.
type LogisticsConfig struct {
	Provider string
}

func LoadLogisticsConfig() LogisticsConfig {
	return LogisticsConfig{
		Provider: GetEnv("LOGISTICS_PROVIDER", "manual"),
	}
}
