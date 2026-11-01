package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeUsernames(t *testing.T) {
	t.Run("splits comma-separated and trims", func(t *testing.T) {
		got := NormalizeUsernames("admin1, admin2 ,admin3")
		assert.Equal(t, []string{"admin1", "admin2", "admin3"}, got)
	})

	t.Run("drops empty entries", func(t *testing.T) {
		got := NormalizeUsernames("admin1,, admin2, ")
		assert.Equal(t, []string{"admin1", "admin2"}, got)
	})

	t.Run("returns nil for blank input", func(t *testing.T) {
		assert.Nil(t, NormalizeUsernames(""))
		assert.Nil(t, NormalizeUsernames("   "))
	})

	t.Run("returns empty for all whitespace after trim", func(t *testing.T) {
		got := NormalizeUsernames(" , , ")
		assert.Equal(t, []string{}, got)
	})

	t.Run("single username without comma", func(t *testing.T) {
		got := NormalizeUsernames("admin1")
		assert.Equal(t, []string{"admin1"}, got)
	})
}
