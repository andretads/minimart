package http

import (
	"encoding/json"
	"minimart/internal/category/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCategory(t *testing.T) {
	t.Run("returns a category by ID", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := do(t, mux, "GET", "/v1/categories/cat-1", "")
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "cat-1", data.ID)
		assert.Equal(t, "Electronics", data.Name)
		assert.Equal(t, "📱", data.Icon)
		assert.Equal(t, 1, data.Version)

		etag := rec.Header().Get("ETag")
		assert.Equal(t, `"1"`, etag)
	})

	t.Run("rejects unknown ID", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := do(t, mux, "GET", "/v1/categories/unknown", "")
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: não encontrada", env.Error.Message)
	})

	t.Run("returns 304 for unchanged If-None-Match", func(t *testing.T) {
		mux, repo := newTestMux(t)

		c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
		_ = repo.Save(t.Context(), c)

		rec := doWithHeaders(t, mux, "GET", "/v1/categories/cat-1", "", map[string]string{
			"If-None-Match": `"1"`,
		})
		assert.Equal(t, http.StatusNotModified, rec.Code)
		assert.Empty(t, rec.Body.String())
	})
}
