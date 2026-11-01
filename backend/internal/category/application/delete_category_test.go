package application

import (
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCategory(t *testing.T) {
	t.Run("deletes category with correct version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID:      created.ID,
			Version: created.Version,
		})
		require.NoError(t, err)

		// Verify it's gone
		_, err = svc.GetCategory(t.Context(), created.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		err := svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID:      "unknown",
			Version: 1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rejects missing version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID: created.ID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("rejects stale version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		// Another writer deletes first
		err = svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID:      created.ID,
			Version: created.Version,
		})
		require.NoError(t, err)

		// Now try with stale version
		err = svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID:      created.ID,
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound) // or ErrCategoryOptimisticLock depending on implementation
	})

	t.Run("deleted category no longer appears in list", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)
		created2, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Books",
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), DeleteCategoryRequest{
			ID:      created2.ID,
			Version: created2.Version,
		})
		require.NoError(t, err)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Electronics", got[0].Name)
	})
}
