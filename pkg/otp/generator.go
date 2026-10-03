package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

type NumericGenerator struct {
	length int
}

func NewNumericGenerator(length int) Generator {
	return &NumericGenerator{
		length: length,
	}
}

func (g *NumericGenerator) Generate() (string, error) {
	var builder strings.Builder

	for range g.length {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate otp digit: %w", err)
		}

		builder.WriteString(n.String())
	}

	return builder.String(), nil
}
