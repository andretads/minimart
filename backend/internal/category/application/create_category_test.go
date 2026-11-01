package application

import (
	"minimart/internal/category/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCategoryRequest()
		req.Name = "   "

		_, err := svc.CreateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Name too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCategoryRequest()
		req.Name = strings.Repeat("a", 101)

		_, err := svc.CreateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNameTooLong)
	})

	t.Run("Icon too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCategoryRequest()
		req.Icon = strings.Repeat("a", 501)

		_, err := svc.CreateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIconTooLong)
	})

	t.Run("Duplicate name is rejected case-insensitively", func(t *testing.T) {
		svc, _ := newTestService(t)

		createTestCategory(t, svc, validCreateCategoryRequest())

		req := validCreateCategoryRequest()
		req.Name = "electronics"

		_, err := svc.CreateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("Category got created with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)

		created := createTestCategory(t, svc, validCreateCategoryRequest())

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Electronics", stored.Name)
		assert.Equal(t, "📱", stored.Icon)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Category without icon gets created", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCategoryRequest()
		req.Name = "Books"
		req.Icon = ""

		created := createTestCategory(t, svc, req)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Books", stored.Name)
		assert.Empty(t, stored.Icon)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Name and icon are trimmed on creation", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCategoryRequest()
		req.Name = "  Electronics  "
		req.Icon = "  📱  "

		created := createTestCategory(t, svc, req)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Electronics", stored.Name)
		assert.Equal(t, "📱", stored.Icon)
		assert.Equal(t, 1, stored.Version)
		assert.False(t, stored.CreatedAt.IsZero())
	})
}
