package money_test

import (
	"testing"

	"komecore/pkg/money"
)

func TestParsePriceToInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasErr   bool
	}{
		{"15000", 1500000, false},
		{"15000.50", 1500050, false},
		{"15000.5", 1500050, false},
		{"10.00", 1000, false},
		{"0.99", 99, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := money.ParsePriceToInt64(tt.input)
		if (err != nil) != tt.hasErr {
			t.Errorf("ParsePriceToInt64(%q) error = %v, expected error: %v", tt.input, err, tt.hasErr)
			continue
		}
		if !tt.hasErr && got != tt.expected {
			t.Errorf("ParsePriceToInt64(%q) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}

func TestFormatInt64ToPrice(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{1500000, "15000.00"},
		{1500050, "15000.50"},
		{99, "0.99"},
		{-1050, "-10.50"},
	}

	for _, tt := range tests {
		got := money.FormatInt64ToPrice(tt.input)
		if got != tt.expected {
			t.Errorf("FormatInt64ToPrice(%d) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
