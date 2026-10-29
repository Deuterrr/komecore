package slug

import (
	"regexp"
	"strings"
)

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

// latinReplacer transliterates common Latin diacritics into plain ASCII equivalents
// prior to regex hyphenation to preserve URL legibility for international text.
var latinReplacer = strings.NewReplacer(
	// Accented vowels (lowercase)
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
	"è", "e", "é", "e", "ê", "e", "ë", "e",
	"ì", "i", "í", "i", "î", "i", "ï", "i",
	"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o",
	"ù", "u", "ú", "u", "û", "u", "ü", "u",

	// Accented vowels (uppercase)
	"À", "a", "Á", "a", "Â", "a", "Ã", "a", "Ä", "a", "Å", "a",
	"È", "e", "É", "e", "Ê", "e", "Ë", "e",
	"Ì", "i", "Í", "i", "Î", "i", "Ï", "i",
	"Ò", "o", "Ó", "o", "Ô", "o", "Õ", "o", "Ö", "o",
	"Ù", "u", "Ú", "u", "Û", "u", "Ü", "u",

	// Consonants and ligatures
	"ñ", "n", "Ñ", "n",
	"ç", "c", "Ç", "c",
	"ß", "ss", "ẞ", "ss",
)

func transliterate(input string) string {
	return latinReplacer.Replace(input)
}

// Make generates a clean URL-friendly slug from an arbitrary input string.
func Make(input string) string {
	s := transliterate(input)
	s = strings.ToLower(s)
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Generate is an alias for Make.
func Generate(input string) string {
	return Make(input)
}

// Generator abstracts slug generation. Kept for backwards compatibility with existing constructors.
type Generator interface {
	Generate(input string) string
}

type generator struct{}

// NewGenerator returns a Generator backed by slug.Make.
func NewGenerator() Generator {
	return generator{}
}

func (generator) Generate(input string) string {
	return Make(input)
}
