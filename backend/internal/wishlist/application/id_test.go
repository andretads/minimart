package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	t.Run("returns a non-empty rfc 4648 base32 token", func(t *testing.T) {
		id := newID()
		assert.NotEmpty(t, id)
		assert.GreaterOrEqual(t, len(id), 26, "at least 128 bits of randomness")
		assert.Regexp(t, `^[A-Z2-7]+$`, id)
	})

	t.Run("is unique across many draws", func(t *testing.T) {
		seen := make(map[string]struct{}, 1000)
		for range 1000 {
			id := newID()
			_, dup := seen[id]
			require.False(t, dup, "duplicate id: %q", id)
			seen[id] = struct{}{}
		}
	})
}
