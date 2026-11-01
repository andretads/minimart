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

func TestCreateCategory(t *testing.T) {
	t.Run("creates a category with name and icon", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "Electronics", got.Name)
		assert.Equal(t, "📱", got.Icon)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("creates a category without icon", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Books",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "Books", got.Name)
		assert.Empty(t, got.Icon)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "   ",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longName := strings.Repeat("a", 101)
		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: longName,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNameTooLong)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		longIcon := strings.Repeat("a", 501)
		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: longIcon,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIconTooLong)
	})

	t.Run("rejects duplicate name case-insensitively", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
		})
		require.NoError(t, err)

		_, err = svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "electronics",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "  Electronics  ",
			Icon: "  📱  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "Electronics", got.Name)
		assert.Equal(t, "📱", got.Icon)
	})
}
