package money

import (
	"fmt"
	"strconv"
	"strings"
)

func FormatFloatToString(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
}

func ParseStringToFloat(val *string) (*float64, error) {
	if val == nil {
		return nil, nil
	}

	str := strings.TrimSpace(*val)
	if str == "" {
		return nil, nil
	}

	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse float: %w", err)
	}

	return &f, nil
}
