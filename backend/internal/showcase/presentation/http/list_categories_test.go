package http

import (
	"encoding/json"
	"minimart/internal/showcase/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCategories(t *testing.T) {
	t.Run("empty storage returns empty list", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := do(t, mux, "GET", "/v1/showcase/categories")
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data []categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Empty(t, data)
	})

	t.Run("returns all categories ordered by name", func(t *testing.T) {
		mux, repo := newTestMux(t)

		repo.SaveCategory(domain.Category{ID: "c1", Name: "Electronics", Icon: "📱"})
		repo.SaveCategory(domain.Category{ID: "c2", Name: "Books", Icon: "📚"})
		repo.SaveCategory(domain.Category{ID: "c3", Name: "Clothing", Icon: "👕"})

		rec := do(t, mux, "GET", "/v1/showcase/categories")
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data []categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		require.Len(t, data, 3)
		assert.Equal(t, "Books", data[0].Name)
		assert.Equal(t, "Clothing", data[1].Name)
		assert.Equal(t, "Electronics", data[2].Name)
	})
}
