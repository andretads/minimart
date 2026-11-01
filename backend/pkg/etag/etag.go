// Package etag provides helpers for formatting and parsing HTTP entity tags (ETags)
// used for optimistic locking. ETags are based on an integer version that is
// bumped on every persisted write.
package etag

import (
	"strconv"
	"strings"
)

// Format returns a strong entity tag for the given version, e.g. `"4"`.
func Format(version int) string {
	return `"` + strconv.Itoa(version) + `"`
}

// Matches reports whether a conditional-request header value (If-Match or
// If-None-Match) selects the given version. It accepts a comma-separated list,
// the weak form `W/"4"`, and the `*` wildcard. An empty header never matches.
func Matches(header string, version int) bool {
	if header == "" {
		return false
	}
	for candidate := range strings.SplitSeq(header, ",") {
		got, wildcard, ok := Parse(candidate)
		if !ok {
			continue
		}
		if wildcard || got == version {
			return true
		}
	}
	return false
}

// Parse parses one entity tag into a version. wildcard is true for `*`; ok
// is false for a malformed tag or a non-positive version.
func Parse(value string) (version int, wildcard, ok bool) {
	value = strings.TrimSpace(value)
	if value == "*" {
		return 0, true, true
	}
	value = strings.TrimPrefix(value, "W/")
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return 0, false, false
	}
	n, err := strconv.Atoi(value[1 : len(value)-1])
	if err != nil || n <= 0 {
		return 0, false, false
	}
	return n, false, true
}
