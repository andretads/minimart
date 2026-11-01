package country

import (
	"strings"
)

// aliases maps lower-cased, trimmed localized country names to their
// ISO-3166 alpha-2 code. Extend this list as additional localized values
// are introduced.
var aliases = map[string]string{
	"brasil":  "BR",
	"brazil":  "BR",
	"brasil.": "BR",
}

// Normalize converts a country value to its canonical ISO-3166 alpha-2 code.
// Unknown values are returned trimmed and upper-cased only when they are
// already two letters; other values are returned trimmed but unchanged, so
// that domain-specific test fixtures (e.g. "C-1234") keep working.
func Normalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if code, ok := aliases[strings.ToLower(value)]; ok {
		return code
	}
	if len(value) == 2 {
		return strings.ToUpper(value)
	}
	return value
}

// Equal reports whether two country values refer to the same country.
func Equal(a, b string) bool {
	return Normalize(a) == Normalize(b)
}
