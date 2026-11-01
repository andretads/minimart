package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	ids := make(map[string]struct{}, 1000)
	for range 1000 {
		id := newID()
		require.NotEmpty(t, id)
		require.GreaterOrEqual(t, len(id), 26)
		assert.Regexp(t, "^[A-Z2-7]+$", id)
		_, exists := ids[id]
		require.False(t, exists, "duplicate ID generated: %s", id)
		ids[id] = struct{}{}
	}
}
