//go:build integration

package test

import (
	"minimart/internal/category/domain"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCategoryIntegration(t *testing.T) {
	svc := newTestService(t)

	t.Run("Category does not exist", func(t *testing.T) {
		_, err := svc.UpdateCategory(t.Context(), validUpdateCategoryRequest("missing", 1))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Blank name is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		req := validUpdateCategoryRequest(created.ID, created.Version)
		req.Name = "   "

		_, err := svc.UpdateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Name too long is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		req := validUpdateCategoryRequest(created.ID, created.Version)
		req.Name = strings.Repeat("a", 101)

		_, err := svc.UpdateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNameTooLong)
	})

	t.Run("Icon too long is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		req := validUpdateCategoryRequest(created.ID, created.Version)
		req.Icon = strings.Repeat("a", 501)

		_, err := svc.UpdateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrIconTooLong)
	})

	t.Run("Duplicate name is rejected", func(t *testing.T) {
		createTestCategory(t, svc, validCreateCategoryRequest())

		booksReq := validCreateCategoryRequest()
		booksReq.Name = "Books"
		booksReq.Icon = "📚"
		books := createTestCategory(t, svc, booksReq)

		req := validUpdateCategoryRequest(books.ID, books.Version)
		req.Name = "electronics"

		_, err := svc.UpdateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryExists)
	})

	t.Run("Missing version is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		req := validUpdateCategoryRequest(created.ID, 0)

		_, err := svc.UpdateCategory(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		_, err := svc.UpdateCategory(t.Context(), validUpdateCategoryRequest(created.ID, created.Version))
		require.NoError(t, err)

		_, err = svc.UpdateCategory(t.Context(), validUpdateCategoryRequest(created.ID, created.Version))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("Stale ETag is rejected", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		_, err := svc.UpdateCategory(t.Context(), validUpdateCategoryRequest(created.ID, 99))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryOptimisticLock)
	})

	t.Run("Category got updated with no problems", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		_, err := svc.UpdateCategory(t.Context(), validUpdateCategoryRequest(created.ID, created.Version))
		require.NoError(t, err)

		stored, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, stored.ID)
		assert.Equal(t, "Gadgets", stored.Name)
		assert.Equal(t, "🔧", stored.Icon)
		assert.Equal(t, 2, stored.Version)
		assert.WithinDuration(t, created.CreatedAt, stored.CreatedAt, time.Second)
	})

	t.Run("Category is updated using the ETag from GET", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		current, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)

		_, err = svc.UpdateCategory(t.Context(), validUpdateCategoryRequest(current.ID, current.Version))
		require.NoError(t, err)

		stored, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, stored.ID)
		assert.Equal(t, "Gadgets", stored.Name)
		assert.Equal(t, "🔧", stored.Icon)
		assert.Equal(t, 2, stored.Version)
		assert.WithinDuration(t, created.CreatedAt, stored.CreatedAt, time.Second)
	})

	t.Run("Name and icon are trimmed on update", func(t *testing.T) {
		created := createTestCategory(t, svc, validCreateCategoryRequest())

		req := validUpdateCategoryRequest(created.ID, created.Version)
		req.Name = "  Gadgets  "
		req.Icon = "  🔧  "

		_, err := svc.UpdateCategory(t.Context(), req)
		require.NoError(t, err)

		stored, err := svc.GetCategory(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, stored.ID)
		assert.Equal(t, "Gadgets", stored.Name)
		assert.Equal(t, "🔧", stored.Icon)
		assert.Equal(t, 2, stored.Version)
		assert.WithinDuration(t, created.CreatedAt, stored.CreatedAt, time.Second)
	})
}
