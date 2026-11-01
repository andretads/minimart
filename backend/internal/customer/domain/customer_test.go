package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCustomer(t *testing.T) {
	t.Run("creates a customer with an id and user id", func(t *testing.T) {
		before := time.Now().UTC()
		c, err := NewCustomer("cust-1", "user-1", "")
		require.NoError(t, err)
		assert.Equal(t, "cust-1", c.ID)
		assert.Equal(t, "user-1", c.UserID)
		assert.Empty(t, c.DisplayName)
		assert.Equal(t, StatusActive, c.Status)
		assert.False(t, c.CreatedAt.Before(before))
		assert.False(t, c.CreatedAt.After(time.Now().UTC()))
	})

	t.Run("rejects an empty user id", func(t *testing.T) {
		_, err := NewCustomer("cust-1", "", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyUserID)
	})

	t.Run("rejects display name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 201)
		_, err := NewCustomer("cust-1", "user-1", longName)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDisplayNameTooLong)
	})

	t.Run("trims display name", func(t *testing.T) {
		c, err := NewCustomer("cust-1", "user-1", "  John  ")
		require.NoError(t, err)
		assert.Equal(t, "John", c.DisplayName)
	})
}
