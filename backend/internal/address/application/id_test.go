//go:build integration

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	t.Run("shape", func(t *testing.T) {
		id := newID()
		require.NotEmpty(t, id)
		assert.GreaterOrEqual(t, len(id), 26) // ≥128 bits in RFC 4648 base32
		assert.Regexp(t, `^[A-Z2-7]+$`, id)
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]struct{}, 1000)
		for range 1000 {
			id := newID()
			_, exists := seen[id]
			require.False(t, exists, "duplicate ID generated: %s", id)
			seen[id] = struct{}{}
		}
	})
}
