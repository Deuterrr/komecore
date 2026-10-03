package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_]+)\}\}`)

func Render(template string, vars map[string]string) (string, error) {
	matches := placeholderPattern.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		key := match[1]

		value, ok := vars[key]
		if !ok {
			return "", fmt.Errorf("missing template variable: %s", key)
		}

		template = strings.ReplaceAll(
			template,
			match[0],
			value,
		)
	}

	return template, nil
}
