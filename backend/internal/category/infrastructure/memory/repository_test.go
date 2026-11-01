package memory

import (
	"minimart/internal/category/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepository(t *testing.T) {
	t.Run("Save and GetByID", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")

		err := repo.Save(t.Context(), c)
		require.NoError(t, err)

		got, err := repo.GetByID(t.Context(), "cat-1")
		require.NoError(t, err)
		assert.Equal(t, c.ID, got.ID)
		assert.Equal(t, c.Name, got.Name)
		assert.Equal(t, c.Icon, got.Icon)
	})

	t.Run("GetByID not found", func(t *testing.T) {
		repo := New()
		_, err := repo.GetByID(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("List returns categories ordered by name", func(t *testing.T) {
		repo := New()
		c1, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		c2, _ := domain.NewCategory("cat-2", "Books", "📚")
		c3, _ := domain.NewCategory("cat-3", "Clothing", "👕")

		_ = repo.Save(t.Context(), c1)
		_ = repo.Save(t.Context(), c2)
		_ = repo.Save(t.Context(), c3)

		got, err := repo.List(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "Books", got[0].Name)
		assert.Equal(t, "Clothing", got[1].Name)
		assert.Equal(t, "Electronics", got[2].Name)
	})

	t.Run("Save rejects duplicate name case-insensitively", func(t *testing.T) {
		repo := New()
		c1, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		c2, _ := domain.NewCategory("cat-2", "electronics", "📱")

		_ = repo.Save(t.Context(), c1)

		err := repo.Save(t.Context(), c2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("Update with optimistic lock", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		c.Name = "Gadgets"
		c.Version = 1
		updated, err := repo.Update(t.Context(), c)
		require.NoError(t, err)
		assert.Equal(t, "Gadgets", updated.Name)
		assert.Equal(t, 2, updated.Version)
	})

	t.Run("Update rejects stale version", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		// First update
		c.Name = "Gadgets"
		c.Version = 1
		_, _ = repo.Update(t.Context(), c)

		// Second update with stale version
		_, err := repo.Update(t.Context(), c)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("Update rejects unknown ID", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		c.Version = 1

		_, err := repo.Update(t.Context(), c)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Update rejects duplicate name", func(t *testing.T) {
		repo := New()
		c1, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		c2, _ := domain.NewCategory("cat-2", "Books", "📚")
		_ = repo.Save(t.Context(), c1)
		_ = repo.Save(t.Context(), c2)

		c2.Name = "electronics"
		c2.Version = 1
		_, err := repo.Update(t.Context(), c2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("Delete with correct version", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		err := repo.Delete(t.Context(), "cat-1", 1)
		require.NoError(t, err)

		_, err = repo.GetByID(t.Context(), "cat-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Delete rejects stale version", func(t *testing.T) {
		repo := New()
		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		err := repo.Delete(t.Context(), "cat-1", 99)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("Delete rejects unknown ID", func(t *testing.T) {
		repo := New()
		err := repo.Delete(t.Context(), "unknown", 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})
}
