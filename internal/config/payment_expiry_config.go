package config

import "strconv"

// PaymentExpiryConfig controls the background payment expiry job.
// The job scans for past-due pending payments and marks them as expired.
type PaymentExpiryConfig struct {
	IntervalMinutes int
	BatchSize       int
	Concurrency     int
}

func LoadPaymentExpiryConfig() PaymentExpiryConfig {
	var (
		intervalMinutes = 1
		batchSize       = 100
		concurrency     = 5
	)

	if v := GetEnv("PAYMENT_EXPIRY_INTERVAL_MINUTES", strconv.Itoa(intervalMinutes)); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			intervalMinutes = parsed
		}
	}

	if v := GetEnv("PAYMENT_EXPIRY_BATCH_SIZE", strconv.Itoa(batchSize)); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}

	if v := GetEnv("PAYMENT_EXPIRY_CONCURRENCY", strconv.Itoa(concurrency)); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			concurrency = parsed
		}
	}

	return PaymentExpiryConfig{
		IntervalMinutes: intervalMinutes,
		BatchSize:       batchSize,
		Concurrency:     concurrency,
	}
}
