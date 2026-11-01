package http

import (
	"encoding/json"
	"minimart/internal/category/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCategories(t *testing.T) {
	t.Run("empty storage returns empty list", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := do(t, mux, "GET", "/v1/categories", "")
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data []categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Empty(t, data)
	})

	t.Run("returns all categories ordered by name", func(t *testing.T) {
		mux, repo := newTestMux(t)

		// Create categories out of order
		_ = repo.Save(t.Context(), func() domain.Category {
			c, _ := domain.NewCategory("cat-1", "Electronics", "📱")
			return c
		}())
		_ = repo.Save(t.Context(), func() domain.Category {
			c, _ := domain.NewCategory("cat-2", "Books", "📚")
			return c
		}())
		_ = repo.Save(t.Context(), func() domain.Category {
			c, _ := domain.NewCategory("cat-3", "Clothing", "👕")
			return c
		}())

		rec := do(t, mux, "GET", "/v1/categories", "")
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
