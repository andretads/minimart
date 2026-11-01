package application

import (
	"minimart/internal/category/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCategories(t *testing.T) {
	t.Run("empty storage returns empty list", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns all categories ordered by name", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)
		_, err = svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Books",
			Icon: "📚",
		})
		require.NoError(t, err)
		_, err = svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Clothing",
			Icon: "👕",
		})
		require.NoError(t, err)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "Books", got[0].Name)
		assert.Equal(t, "Clothing", got[1].Name)
		assert.Equal(t, "Electronics", got[2].Name)
	})
}
