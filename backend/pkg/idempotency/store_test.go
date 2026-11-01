package idempotency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFingerprint(t *testing.T) {
	t.Run("is stable for the same parts", func(t *testing.T) {
		got := Fingerprint("a", "b", "c")
		assert.NotEmpty(t, got)
		assert.Equal(t, got, Fingerprint("a", "b", "c"))
	})

	t.Run("is order sensitive", func(t *testing.T) {
		assert.NotEqual(t, Fingerprint("a", "b"), Fingerprint("b", "a"))
	})

	t.Run("length prefixes parts so concatenation is unambiguous", func(t *testing.T) {
		assert.NotEqual(t, Fingerprint("ab", "c"), Fingerprint("a", "bc"))
	})

	t.Run("differs when any part differs", func(t *testing.T) {
		assert.NotEqual(t, Fingerprint("a", "b"), Fingerprint("a", "z"))
	})
}
