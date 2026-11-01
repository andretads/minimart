package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	t.Run("creates a category with name and icon", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCategoryRequest(t, "Electronics", "📱")

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.NotEmpty(t, data.ID)
		assert.Equal(t, "Electronics", data.Name)
		assert.Equal(t, "📱", data.Icon)
		assert.Equal(t, 1, data.Version)

		etag := rec.Header().Get("ETag")
		assert.Equal(t, `"1"`, etag)
	})

	t.Run("creates a category without icon", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCategoryRequest(t, "Books", "")

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Books", data.Name)
		assert.Empty(t, data.Icon)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCategoryRequest(t, "   ", "📱")

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: nome é obrigatório", env.Error.Message)
	})

	t.Run("rejects name too long", func(t *testing.T) {
		mux, _ := newTestMux(t)
		longName := strings.Repeat("a", 101)
		body := buildCreateCategoryRequest(t, longName, "📱")

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: nome muito longo", env.Error.Message)
	})

	t.Run("rejects icon too long", func(t *testing.T) {
		mux, _ := newTestMux(t)
		longIcon := strings.Repeat("a", 501)
		body := buildCreateCategoryRequest(t, "Electronics", longIcon)

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: ícone muito longo", env.Error.Message)
	})

	t.Run("rejects duplicate name case-insensitively", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCategoryRequest(t, "Electronics", "📱")
		rec := do(t, mux, "POST", "/v1/categories", body)
		require.Equal(t, http.StatusCreated, rec.Code)

		rec = do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusConflict, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "category: já existe", env.Error.Message)
	})

	t.Run("trims name and icon whitespace", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCategoryRequest(t, "  Electronics  ", "  📱  ")

		rec := do(t, mux, "POST", "/v1/categories", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data categoryResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Electronics", data.Name)
		assert.Equal(t, "📱", data.Icon)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := do(t, mux, "POST", "/v1/categories", "{invalid")
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "corpo JSON inválido", env.Error.Message)
	})
}
