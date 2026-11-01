package http

import (
	"minimart/internal/category/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCategory(t *testing.T) {
	t.Run("deletes category with correct If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := doWithHeaders(t, mux, "DELETE", "/v1/categories/cat-1", "", map[string]string{
			"If-Match": `"1"`,
		})
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())

		// Verify it's gone
		rec = do(t, mux, "GET", "/v1/categories/cat-1", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doWithHeaders(t, mux, "DELETE", "/v1/categories/unknown", "", map[string]string{
			"If-Match": `"1"`,
		})
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: não encontrada", env.Error.Message)
	})

	t.Run("rejects missing If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := do(t, mux, "DELETE", "/v1/categories/cat-1", "")
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match obrigatório", env.Error.Message)
	})

	t.Run("rejects malformed If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := doWithHeaders(t, mux, "DELETE", "/v1/categories/cat-1", "", map[string]string{
			"If-Match": `not-a-tag`,
		})
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match inválido", env.Error.Message)
	})

	t.Run("rejects wildcard If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := doWithHeaders(t, mux, "DELETE", "/v1/categories/cat-1", "", map[string]string{
			"If-Match": `*`,
		})
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match inválido", env.Error.Message)
	})

	t.Run("rejects stale If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		// Delete once
		doWithHeaders(t, mux, "DELETE", "/v1/categories/cat-1", "", map[string]string{
			"If-Match": `"1"`,
		})

		// Try to delete again with stale version
		rec := doWithHeaders(t, mux, "DELETE", "/v1/categories/cat-1", "", map[string]string{
			"If-Match": `"1"`,
		})
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		// Could be ErrCategoryNotFound or ErrCategoryOptimisticLock
		assert.Contains(t, []string{"category: não encontrada", "category: conflito de bloqueio"}, env.Error.Message)
	})
}
