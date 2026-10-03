package slug_test

import (
	"testing"

	"komecore/pkg/slug"
)

func TestSlugGenerator(t *testing.T) {
	gen := slug.NewGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"  Product #1 - Special Edition!  ", "product-1-special-edition"},
		{"electronics & gadgets", "electronics-gadgets"},
		{"multiple---hyphens___and spaces", "multiple-hyphens-and-spaces"},
	}

	for _, tt := range tests {
		got := gen.Generate(tt.input)
		if got != tt.expected {
			t.Errorf("Generate(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
