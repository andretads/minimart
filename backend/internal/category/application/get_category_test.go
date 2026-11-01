package application

import (
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCategory(t *testing.T) {
	t.Run("retrieves a category by ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		created, err := svc.CreateCategory(t.Context(), CreateCategoryRequest{
			Name: "Electronics",
			Icon: "📱",
		})
		require.NoError(t, err)

		got, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Electronics", got.Name)
		assert.Equal(t, "📱", got.Icon)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, metrics.Noop{})

		_, err := svc.GetCategory(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})
}
