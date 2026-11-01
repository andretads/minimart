package etag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	assert.Equal(t, `"1"`, Format(1))
	assert.Equal(t, `"42"`, Format(42))
}

func TestMatches(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		version int
		want    bool
	}{
		{name: "empty header never matches", header: "", version: 1, want: false},
		{name: "exact match", header: `"1"`, version: 1, want: true},
		{name: "weak match", header: `W/"1"`, version: 1, want: true},
		{name: "wildcard", header: `*`, version: 7, want: true},
		{name: "mismatch", header: `"2"`, version: 1, want: false},
		{name: "list with one match", header: `"2", "1"`, version: 1, want: true},
		{name: "list with no match", header: `"3", "2"`, version: 1, want: false},
		{name: "malformed entries are skipped", header: `garbage, "1"`, version: 1, want: true},
		{name: "unquoted number is not a tag", header: `1`, version: 1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Matches(tt.header, tt.version))
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantVer   int
		wantWild  bool
		wantValid bool
	}{
		{name: "strong tag", value: `"5"`, wantVer: 5, wantValid: true},
		{name: "weak tag", value: `W/"5"`, wantVer: 5, wantValid: true},
		{name: "surrounding space", value: `  "5"  `, wantVer: 5, wantValid: true},
		{name: "wildcard", value: `*`, wantWild: true, wantValid: true},
		{name: "unquoted", value: `5`, wantValid: false},
		{name: "empty tag", value: `""`, wantValid: false},
		{name: "zero version", value: `"0"`, wantValid: false},
		{name: "negative version", value: `"-1"`, wantValid: false},
		{name: "not a number", value: `"abc"`, wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, wildcard, ok := Parse(tt.value)
			assert.Equal(t, tt.wantValid, ok)
			assert.Equal(t, tt.wantVer, version)
			assert.Equal(t, tt.wantWild, wildcard)
		})
	}
}
