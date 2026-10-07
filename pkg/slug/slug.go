package slug

import (
	"regexp"
	"strings"
)

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)

// Make generates a clean URL-friendly slug from an arbitrary input string.
func Make(input string) string {
	s := strings.ToLower(input)
	s = nonAlphaNumRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Generate is an alias for Make.
func Generate(input string) string {
	return Make(input)
}

// Generator abstracts slug generation. Kept for backwards compatibility with existing usecase constructors.
type Generator interface {
	Generate(input string) string
}

type generatorImpl struct{}

// NewGenerator returns a Generator backed by slug.Make.
func NewGenerator() Generator {
	return generatorImpl{}
}

func (generatorImpl) Generate(input string) string {
	return Make(input)
}
