package slug

import (
	"regexp"
	"strings"
)

type generatorImpl struct{}

func NewGenerator() Generator {
	return &generatorImpl{}
}

func (g *generatorImpl) Generate(input string) string {
	s := strings.ToLower(input)

	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")

	return s
}
