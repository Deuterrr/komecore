package logger

import "strings"

// forbiddenFields is the authoritative set of field
// keys that must never appear in any log entry.
// Keys are stored in lowercase and matched
// case-insensitively at runtime.
//
// This list is derived from the project logging standards.
// See docs/logging-standards.md for the full policy.
var forbiddenFields = map[string]struct{}{
	"password":             {},
	"access_token":         {},
	"refresh_token":        {},
	"authorization_header": {},
	"authorization":        {},
	"otp":                  {},
	"cvv":                  {},
	"credit_card_number":   {},
	"card_number":          {},
	"session_cookie":       {},
	"cookie":               {},
	"secret":               {},
	"private_key":          {},
	"api_key":              {},
	"api_secret":           {},
}

// Redact returns a copy of fields with sensitive
// values replaced by "[REDACTED]".
//
// This should be called before passing any
// user-supplied or request-derived data to a log method.
//
// It does NOT mutate the input slice.
//
// Usage:
//
//	safe := applogger.Redact(fields)
//	log.Info(ctx, "user_updated", safe...)
//
// Note: Redact only protects against explicitly named keys.
// Never log raw request bodies, full structs,
// or fmt.Sprintf("%+v", user) as these can leak
// sensitive data even after redaction.
func Redact(fields []Field) []Field {
	out := make([]Field, len(fields))
	for i, f := range fields {
		if _, forbidden := forbiddenFields[strings.ToLower(f.Key)]; forbidden {
			out[i] = Field{Key: f.Key, Value: "[REDACTED]"}
		} else {
			out[i] = f
		}
	}
	return out
}

// IsForbidden reports whether a field key is
// in the forbidden list.
//
// Useful for validation in tests or custom log handlers.
func IsForbidden(key string) bool {
	_, ok := forbiddenFields[strings.ToLower(key)]
	return ok
}
