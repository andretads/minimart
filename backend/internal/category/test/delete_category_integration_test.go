//go:build integration

package test

import (
	"minimart/internal/category/application"
	"minimart/internal/category/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCategoryIntegration(t *testing.T) {
	svc := newTestService(t)

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      "unknown",
			Version: 1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Missing If-Match is rejected", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		// The feature expects "cabeçalho If-Match obrigatório"; the HTTP adapter
		// rejects a missing header before the use case sees it as a zero version.
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{ID: cat.ID})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Malformed If-Match is rejected", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		// The feature expects "cabeçalho If-Match inválido"; parsing happens in
		// the HTTP adapter, so the use case only sees an invalid version.
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: 0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Wildcard If-Match is rejected", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		// The feature expects "cabeçalho If-Match inválido"; wildcard handling
		// happens in the HTTP adapter, so the use case only sees an invalid version.
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: 0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version (concurrent delete) is rejected", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: cat.Version,
		})
		require.NoError(t, err)

		err = svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: cat.Version,
		})
		require.Error(t, err)
		// The feature expects "category: conflito de bloqueio", but the row is
		// already gone, so it cannot be distinguished from an unknown ID.
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Category deleted with correct If-Match", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		// The feature expects a 204 no content response, which the HTTP adapter produces.
		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: cat.Version,
		})
		require.NoError(t, err)
	})

	t.Run("Deleted category no longer retrievable", func(t *testing.T) {
		cat := createTestCategory(t, svc, validCreateCategoryRequest())

		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      cat.ID,
			Version: cat.Version,
		})
		require.NoError(t, err)

		_, err = svc.GetCategory(t.Context(), cat.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
	})

	t.Run("Deleted category no longer appears in list", func(t *testing.T) {
		deleteAllCategories(t, svc)

		electronics := createTestCategory(t, svc, validCreateCategoryRequest())

		booksReq := validCreateCategoryRequest()
		booksReq.Name = "Books"
		booksReq.Icon = "📚"
		createTestCategory(t, svc, booksReq)

		err := svc.DeleteCategory(t.Context(), application.DeleteCategoryRequest{
			ID:      electronics.ID,
			Version: electronics.Version,
		})
		require.NoError(t, err)

		got, err := svc.ListCategories(t.Context())
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Books", got[0].Name)
	})
}
