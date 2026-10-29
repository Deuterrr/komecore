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
		{"Crème Brûlée", "creme-brulee"},
		{"São Paulo Roast", "sao-paulo-roast"},
		{"Jalapeño Popper", "jalapeno-popper"},
		{"München Bier 500ml", "munchen-bier-500ml"},
		{"El Niño Special", "el-nino-special"},
		{"Großhandel Express", "grosshandel-express"},
		{"Français Café", "francais-cafe"},
		{"Éclair au Chocolat", "eclair-au-chocolat"},
		{"àáâãäå àáâãäå", "aaaaaa-aaaaaa"},
		{"èéêë èéêë", "eeee-eeee"},
		{"ìíîï ìíîï", "iiii-iiii"},
		{"òóôõö òóôõö", "ooooo-ooooo"},
		{"ùúûü ùúûü", "uuuu-uuuu"},
		{"ÀÁÂÃÄÅ ÈÉÊË ÌÍÎÏ ÒÓÔÕÖ ÙÚÛÜ Ñ Ç", "aaaaaa-eeee-iiii-ooooo-uuuu-n-c"},
	}

	for _, tt := range tests {
		// Test interface implementation
		got := gen.Generate(tt.input)
		if got != tt.expected {
			t.Errorf("gen.Generate(%q) = %q, expected %q", tt.input, got, tt.expected)
		}

		// Test pure function
		pureGot := slug.Make(tt.input)
		if pureGot != tt.expected {
			t.Errorf("slug.Make(%q) = %q, expected %q", tt.input, pureGot, tt.expected)
		}
	}
}
