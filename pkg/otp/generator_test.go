package otp_test

import (
	"testing"

	"komecore/pkg/otp"
)

func TestNumericGenerator(t *testing.T) {
	lengths := []int{4, 6, 8}

	for _, l := range lengths {
		gen := otp.NewNumericGenerator(l)
		code, err := gen.Generate()
		if err != nil {
			t.Fatalf("unexpected error generating %d-digit otp: %v", l, err)
		}
		if len(code) != l {
			t.Errorf("expected length %d, got %d (code: %s)", l, len(code), code)
		}
		for _, ch := range code {
			if ch < '0' || ch > '9' {
				t.Errorf("expected numeric character, got %c", ch)
			}
		}
	}
}
