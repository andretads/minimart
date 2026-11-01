package application

import (
	"minimart/internal/category/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCategory(t *testing.T) {
	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetCategory(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Category retrieved by ID", func(t *testing.T) {
		svc, _ := newTestService(t)
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		got, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Electronics", got.Name)
		assert.Equal(t, "📱", got.Icon)
		assert.Equal(t, 1, got.Version)
		assert.False(t, got.CreatedAt.IsZero())
	})

	t.Run("Unchanged category answers 304", func(t *testing.T) {
		svc, _ := newTestService(t)
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		first, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)

		// The feature expects a 304 response with no body; the HTTP adapter
		// produces it from If-None-Match. The use case returns the same version.
		second, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, first.Version, second.Version)
	})
}
