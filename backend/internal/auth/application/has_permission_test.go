package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPermission(t *testing.T) {
	svc := &Service{}

	t.Run("returns true when permission exists", func(t *testing.T) {
		assert.True(t, svc.HasPermission([]string{"catalog:read", "orders:write"}, "catalog:read"))
	})

	t.Run("returns false when permission missing", func(t *testing.T) {
		assert.False(t, svc.HasPermission([]string{"catalog:read"}, "orders:write"))
	})

	t.Run("returns false for empty list", func(t *testing.T) {
		assert.False(t, svc.HasPermission([]string{}, "catalog:read"))
	})
}

func TestHasAnyPermission(t *testing.T) {
	svc := &Service{}

	t.Run("returns true when any permission matches", func(t *testing.T) {
		assert.True(t, svc.HasAnyPermission([]string{"catalog:read"}, "orders:write", "catalog:read"))
	})

	t.Run("returns false when no permissions match", func(t *testing.T) {
		assert.False(t, svc.HasAnyPermission([]string{"catalog:read"}, "orders:write", "payments:read"))
	})

	t.Run("returns false for empty required list", func(t *testing.T) {
		assert.False(t, svc.HasAnyPermission([]string{"catalog:read"}))
	})

	t.Run("returns false for empty permissions list", func(t *testing.T) {
		assert.False(t, svc.HasAnyPermission([]string{}, "catalog:read"))
	})
}
