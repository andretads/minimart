package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategory(t *testing.T) {
	t.Run("creates a category with name and icon", func(t *testing.T) {
		c, err := NewCategory("cat-1", "Electronics", "📱")
		require.NoError(t, err)
		assert.Equal(t, "cat-1", c.ID)
		assert.Equal(t, "Electronics", c.Name)
		assert.Equal(t, "📱", c.Icon)
		assert.Equal(t, 1, c.Version)
		assert.False(t, c.CreatedAt.IsZero())
		assert.False(t, c.UpdatedAt.IsZero())
	})

	t.Run("creates a category without icon", func(t *testing.T) {
		c, err := NewCategory("cat-1", "Books", "")
		require.NoError(t, err)
		assert.Equal(t, "Books", c.Name)
		assert.Empty(t, c.Icon)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := NewCategory("cat-1", "   ", "📱")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyName)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 101)
		_, err := NewCategory("cat-1", longName, "📱")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameTooLong)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		longIcon := strings.Repeat("a", 501)
		_, err := NewCategory("cat-1", "Electronics", longIcon)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIconTooLong)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		c, err := NewCategory("cat-1", "  Electronics  ", "  📱  ")
		require.NoError(t, err)
		assert.Equal(t, "Electronics", c.Name)
		assert.Equal(t, "📱", c.Icon)
	})
}

func TestCategory_UpdateNameAndIcon(t *testing.T) {
	t.Run("updates name and icon", func(t *testing.T) {
		c, _ := NewCategory("cat-1", "Electronics", "📱")
		err := c.UpdateNameAndIcon("Gadgets", "🔧")
		require.NoError(t, err)
		assert.Equal(t, "Gadgets", c.Name)
		assert.Equal(t, "🔧", c.Icon)
		assert.False(t, c.UpdatedAt.IsZero())
		assert.Equal(t, 1, c.Version) // version not bumped by domain
	})

	t.Run("rejects empty name", func(t *testing.T) {
		c, _ := NewCategory("cat-1", "Electronics", "📱")
		err := c.UpdateNameAndIcon("   ", "🔧")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyName)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		c, _ := NewCategory("cat-1", "Electronics", "📱")
		longName := strings.Repeat("a", 101)
		err := c.UpdateNameAndIcon(longName, "🔧")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameTooLong)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		c, _ := NewCategory("cat-1", "Electronics", "📱")
		longIcon := strings.Repeat("a", 501)
		err := c.UpdateNameAndIcon("Gadgets", longIcon)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIconTooLong)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		c, _ := NewCategory("cat-1", "Electronics", "📱")
		err := c.UpdateNameAndIcon("  Gadgets  ", "  🔧  ")
		require.NoError(t, err)
		assert.Equal(t, "Gadgets", c.Name)
		assert.Equal(t, "🔧", c.Icon)
	})
}
