package config

import "strconv"

// PaymentSyncConfig controls the background payment reconciliation job.
// The job polls Midtrans for any pending gateway payments that were missed
// by webhook delivery (e.g. when the service was unreachable).
type PaymentSyncConfig struct {
	// IntervalMinutes is how often the reconciliation job runs.
	IntervalMinutes int

	// LookbackHours is how far back in time the job scans for
	// pending payments. Payments older than this window are skipped
	// (they are almost certainly expired on Midtrans's side already).
	LookbackHours int
}

func LoadPaymentSyncConfig() PaymentSyncConfig {
	intervalMinutes := 30
	if v := GetEnv("PAYMENT_SYNC_INTERVAL_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			intervalMinutes = parsed
		}
	}

	lookbackHours := 24
	if v := GetEnv("PAYMENT_SYNC_LOOKBACK_HOURS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			lookbackHours = parsed
		}
	}

	return PaymentSyncConfig{
		IntervalMinutes: intervalMinutes,
		LookbackHours:   lookbackHours,
	}
}
