package application

import (
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCategory(t *testing.T) {
	t.Run("updates category name and icon", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Gadgets",
			Icon:    "🔧",
			Version: created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Gadgets", updated.Name)
		assert.Equal(t, "🔧", updated.Icon)
		assert.Equal(t, 2, updated.Version)
		assert.Equal(t, created.CreatedAt, updated.CreatedAt) // created_at unchanged
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      "unknown",
			Name:    "Gadgets",
			Version: 1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "   ",
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		longName := strings.Repeat("a", 101)
		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    longName,
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNameTooLong)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		longIcon := strings.Repeat("a", 501)
		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Gadgets",
			Icon:    longIcon,
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIconTooLong)
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
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

		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created2.ID,
			Name:    "electronics",
			Version: created2.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("rejects missing version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:   created.ID,
			Name: "Gadgets",
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

		// Another writer updates first
		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Gadgets",
			Version: created.Version,
		})
		require.NoError(t, err)

		// Now try with stale version
		_, err = svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "Gadgets 2",
			Version: created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateCategory(t.Context(), UpdateCategoryRequest{
			ID:      created.ID,
			Name:    "  Gadgets  ",
			Icon:    "  🔧  ",
			Version: created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Gadgets", updated.Name)
		assert.Equal(t, "🔧", updated.Icon)
	})
}
