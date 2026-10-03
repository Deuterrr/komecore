package domain

import (
	"testing"
)

func TestPaymentMethod_CalculateFee(t *testing.T) {
	maxFee10k := int64(10000)
	maxFee5k := int64(5000)

	tests := []struct {
		name     string
		method   PaymentMethod
		amount   int64
		expected int64
	}{
		{
			name: "flat fee ignores amount and max fee",
			method: PaymentMethod{
				FeeType:  FeeTypeFlat,
				FeeFixed: 4000,
				FeeMax:   &maxFee10k,
			},
			amount:   500000,
			expected: 4000,
		},
		{
			name: "percentage fee without max fee",
			method: PaymentMethod{
				FeeType:       FeeTypePercentage,
				FeePercentage: 0.02,
				FeeMax:        nil,
			},
			amount:   1000000,
			expected: 20000,
		},
		{
			name: "percentage fee below max fee",
			method: PaymentMethod{
				FeeType:       FeeTypePercentage,
				FeePercentage: 0.02,
				FeeMax:        &maxFee10k,
			},
			amount:   200000,
			expected: 4000,
		},
		{
			name: "percentage fee exceeding max fee is capped",
			method: PaymentMethod{
				FeeType:       FeeTypePercentage,
				FeePercentage: 0.02,
				FeeMax:        &maxFee10k,
			},
			amount:   1000000,
			expected: 10000,
		},
		{
			name: "mixed fee with percentage portion exceeding max fee",
			method: PaymentMethod{
				FeeType:       FeeTypeMixed,
				FeeFixed:      2000,
				FeePercentage: 0.02,
				FeeMax:        &maxFee5k,
			},
			amount:   1000000,
			expected: 2000 + 5000, // 2000 fixed + 5000 capped percentage
		},
		{
			name: "mixed fee with percentage portion below max fee",
			method: PaymentMethod{
				FeeType:       FeeTypeMixed,
				FeeFixed:      2000,
				FeePercentage: 0.02,
				FeeMax:        &maxFee5k,
			},
			amount:   100000,
			expected: 2000 + 2000, // 2000 fixed + 2000 percentage
		},
		{
			name: "percentage fee with zero max fee is not capped",
			method: PaymentMethod{
				FeeType:       FeeTypePercentage,
				FeePercentage: 0.02,
				FeeMax:        func() *int64 { z := int64(0); return &z }(),
			},
			amount:   1000000,
			expected: 20000,
		},
		{
			name: "fee bps with half-up rounding - rounds up",
			method: PaymentMethod{
				FeeType: FeeTypePercentage,
				FeeBps:  200, // 2.00%
			},
			amount:   99999,
			expected: 2000, // (99999 * 200 + 5000) / 10000 = 2000.48 -> 2000
		},
		{
			name: "fee bps with half-up rounding - rounds down below halfway",
			method: PaymentMethod{
				FeeType: FeeTypePercentage,
				FeeBps:  200, // 2.00%
			},
			amount:   24,
			expected: 0, // (24 * 200 + 5000) / 10000 = 9800 / 10000 = 0
		},
		{
			name: "fee bps with half-up rounding - exactly half rounds up",
			method: PaymentMethod{
				FeeType: FeeTypePercentage,
				FeeBps:  200, // 2.00%
			},
			amount:   25,
			expected: 1, // (25 * 200 + 5000) / 10000 = 10000 / 10000 = 1
		},
		{
			name: "fee bps capped by fee max",
			method: PaymentMethod{
				FeeType: FeeTypePercentage,
				FeeBps:  200, // 2.00%
				FeeMax:  &maxFee5k,
			},
			amount:   1000000,
			expected: 5000,
		},
		{
			name: "mixed fee with fee bps and fixed fee",
			method: PaymentMethod{
				FeeType:  FeeTypeMixed,
				FeeFixed: 1500,
				FeeBps:   150, // 1.50%
			},
			amount:   100000,
			expected: 1500 + 1500, // 3000
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.method.CalculateFee(tc.amount)
			if actual != tc.expected {
				t.Errorf("CalculateFee(%d) = %d, want %d", tc.amount, actual, tc.expected)
			}
		})
	}
}

func TestPaymentMethod_Validate_FeeMax(t *testing.T) {
	negativeFeeMax := int64(-100)
	positiveFeeMax := int64(5000)

	method := PaymentMethod{
		Name:          "GoPay",
		Code:          "gopay",
		Provider:      "midtrans",
		Type:          TypeEWallet,
		FeeType:       FeeTypePercentage,
		FeePercentage: 0.02,
		FeeMax:        &negativeFeeMax,
	}

	if err := method.Validate(); err != ErrInvalidFeeMax {
		t.Errorf("Validate() with negative FeeMax = %v, want %v", err, ErrInvalidFeeMax)
	}

	method.FeeMax = &positiveFeeMax
	if err := method.Validate(); err != nil {
		t.Errorf("Validate() with valid FeeMax = %v, want nil", err)
	}

	method.FeeMax = nil
	if err := method.Validate(); err != nil {
		t.Errorf("Validate() with nil FeeMax = %v, want nil", err)
	}
}
