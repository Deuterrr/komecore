package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Numeric generates a cryptographically random numeric OTP string of specified digit length.
func Numeric(length int) (string, error) {
	var builder strings.Builder

	for range length {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate otp digit: %w", err)
		}

		builder.WriteString(n.String())
	}

	return builder.String(), nil
}

// Generator abstracts OTP generation for components that need configurable length or mocking.
type Generator interface {
	Generate() (string, error)
}

// NumericGenerator produces numeric OTP codes of fixed length.
type NumericGenerator struct {
	length int
}

// NewNumericGenerator creates a Generator producing OTPs of the specified length.
func NewNumericGenerator(length int) Generator {
	return &NumericGenerator{
		length: length,
	}
}

// Generate implements Generator.
func (g *NumericGenerator) Generate() (string, error) {
	return Numeric(g.length)
}
