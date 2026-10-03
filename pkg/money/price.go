package money

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePriceToInt64 parses a decimal price string (e.g. "15000.50" or "15000")
// into its sub-unit integer representation (e.g. cents/rupiah integer 1500050).
func ParsePriceToInt64(val string) (int64, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, fmt.Errorf("empty price value")
	}

	parts := strings.Split(val, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid price format: %s", val)
	}

	intPart := parts[0]
	decPart := "00"

	if len(parts) == 2 {
		decPart = parts[1]
	}

	if len(decPart) == 1 {
		decPart += "0"
	} else if len(decPart) > 2 {
		decPart = decPart[:2]
	}

	combined := intPart + decPart

	result, err := strconv.ParseInt(combined, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	return result, nil
}

// FormatInt64ToPrice converts a sub-unit integer representation (e.g. 1500050)
// back to a standard decimal string (e.g. "15000.50").
func FormatInt64ToPrice(val int64) string {
	sign := ""
	if val < 0 {
		sign = "-"
		val = -val
	}

	intPart := val / 100
	decPart := val % 100

	return fmt.Sprintf("%s%d.%02d", sign, intPart, decPart)
}
