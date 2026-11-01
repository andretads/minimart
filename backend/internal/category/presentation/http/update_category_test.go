package http

import (
	"encoding/json"
	"minimart/internal/category/domain"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCategory(t *testing.T) {
	t.Run("updates category with If-Match header", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		rec := doWithHeaders(t, mux, "PATCH", "/v1/categories/cat-1", body, map[string]string{
			"If-Match": `"1"`,
		})
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Gadgets", data.Name)
		assert.Equal(t, "🔧", data.Icon)
		assert.Equal(t, 2, data.Version)

		etag := rec.Header().Get("ETag")
		assert.Equal(t, `"2"`, etag)
	})

	t.Run("updates category with version in body", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Gadgets", data.Name)
		assert.Equal(t, "🔧", data.Icon)
		assert.Equal(t, 2, data.Version)
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		mux, _ := newTestMux(t)

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/unknown", body)
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: não encontrada", env.Error.Message)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "   ", "🔧", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: nome é obrigatório", env.Error.Message)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		longName := strings.Repeat("a", 101)
		body := buildUpdateCategoryRequest(t, longName, "🔧", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: nome muito longo", env.Error.Message)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		longIcon := strings.Repeat("a", 501)
		body := buildUpdateCategoryRequest(t, "Gadgets", longIcon, 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: ícone muito longo", env.Error.Message)
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c1, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		c2, _ := domain.NewCategory("cat-2", "Books", "📚")
		_ = repo.Save(t.Context(), c1)
		_ = repo.Save(t.Context(), c2)

		body := buildUpdateCategoryRequest(t, "electronics", "🔧", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-2", body)
		env := assertJSONResponse(t, rec, http.StatusConflict, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: já existe", env.Error.Message)
	})

	t.Run("rejects missing version", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		req := updateCategoryRequest{Name: "Gadgets", Icon: "🔧"}
		body, _ := json.Marshal(req)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", string(body))
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: versão deve ser maior que zero", env.Error.Message)
	})

	t.Run("rejects stale If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		// Update once to bump version
		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		_ = do(t, mux, "PATCH", "/v1/categories/cat-1", body)

		// Try with stale version
		body = buildUpdateCategoryRequest(t, "Gadgets 2", "🔌", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusConflict, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: conflito de bloqueio", env.Error.Message)
	})

	t.Run("rejects stale If-Match header", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 99)
		rec := doWithHeaders(t, mux, "PATCH", "/v1/categories/cat-1", body, map[string]string{
			"If-Match": `"99"`,
		})
		env := assertJSONResponse(t, rec, http.StatusConflict, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: conflito de bloqueio", env.Error.Message)
	})

	t.Run("rejects malformed If-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		rec := doWithHeaders(t, mux, "PATCH", "/v1/categories/cat-1", body, map[string]string{
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

		body := buildUpdateCategoryRequest(t, "Gadgets", "🔧", 1)
		rec := doWithHeaders(t, mux, "PATCH", "/v1/categories/cat-1", body, map[string]string{
			"If-Match": `*`,
		})
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match inválido", env.Error.Message)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		body := buildUpdateCategoryRequest(t, "  Gadgets  ", "  🔧  ", 1)
		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", body)
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Gadgets", data.Name)
		assert.Equal(t, "🔧", data.Icon)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := do(t, mux, "PATCH", "/v1/categories/cat-1", "{invalid")
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "corpo JSON inválido", env.Error.Message)
	})
}
