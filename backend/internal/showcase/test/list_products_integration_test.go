//go:build integration

package test

import (
	"encoding/json"
	"fmt"
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/postgres"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	showcasehttp "minimart/internal/showcase/presentation/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProductsIntegration(t *testing.T) {
	readPool, _ := mustPools(t)
	repo := postgres.NewRepository(readPool)
	svc := application.NewService(repo, stubSigner{}, time.Hour)
	handlers := showcasehttp.NewHandlers(svc)
	mux := http.NewServeMux()
	handlers.Register(mux)
	pool := readPool

	t.Run("Empty catalog returns an empty paginated list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id=nonexistent-cat", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool  `json:"success"`
			Data    []any `json:"data"`
			Meta    struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
				PageSize   int    `json:"page_size"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		assert.Empty(t, env.Data)
		assert.False(t, env.Meta.HasMore)
		assert.Empty(t, env.Meta.NextCursor)
	})

	t.Run("Seeded catalog returns products newest-first with cursor metadata", func(t *testing.T) {
		cat := "cat-list-seeded"
		seedCategory(t, pool, cat, cat)
		now := time.Now().UTC()
		for i, name := range []string{"Product A", "Product B", "Product C"} {
			seedProduct(t, pool, domain.Product{ID: fmt.Sprintf("lp-seeded-%d", i), Name: name, PriceCents: int64((i + 1) * 1000), CategoryID: cat}, now.Add(time.Duration(i)*time.Second))
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id="+cat, nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				Name string `json:"name"`
			} `json:"data"`
			Meta struct {
				PageSize int  `json:"page_size"`
				HasMore  bool `json:"has_more"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		require.Len(t, env.Data, 3)
		assert.Equal(t, "Product C", env.Data[0].Name)
		assert.Equal(t, 20, env.Meta.PageSize)
		assert.False(t, env.Meta.HasMore)
	})

	t.Run("Products with images expose the primary image URL", func(t *testing.T) {
		cat := "cat-list-img"
		seedCategory(t, pool, cat, cat)
		now := time.Now().UTC()
		seedProduct(t, pool, domain.Product{ID: "lp-img-1", Name: "Product A", PriceCents: 1000, CategoryID: cat}, now)
		seedImage(t, pool, domain.ProductImage{ID: "lp-img-i1", ProductID: "lp-img-1", ObjectKey: "key-1", Position: 1})
		seedImage(t, pool, domain.ProductImage{ID: "lp-img-i2", ProductID: "lp-img-1", ObjectKey: "key-2", Position: 2})

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id="+cat, nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				ImageURL string `json:"image_url"`
				Images   []any  `json:"images"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data, 1)
		assert.NotEmpty(t, env.Data[0].ImageURL)
		assert.Nil(t, env.Data[0].Images)
	})

	t.Run("Page_size limits number of items returned", func(t *testing.T) {
		cat := "cat-list-page"
		seedCategory(t, pool, cat, cat)
		now := time.Now().UTC()
		for i := range 3 {
			seedProduct(t, pool, domain.Product{ID: fmt.Sprintf("lp-page-%d", i), Name: fmt.Sprintf("Product %d", i), PriceCents: 1000, CategoryID: cat}, now.Add(time.Duration(i)*time.Second))
		}

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id="+cat+"&page_size=2", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool  `json:"success"`
			Data    []any `json:"data"`
			Meta    struct {
				PageSize   int    `json:"page_size"`
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data, 2)
		assert.Equal(t, 2, env.Meta.PageSize)
		assert.True(t, env.Meta.HasMore)
		assert.NotEmpty(t, env.Meta.NextCursor)
	})

	t.Run("Cursor advances to the next page", func(t *testing.T) {
		cat := "cat-list-cursor"
		seedCategory(t, pool, cat, cat)
		now := time.Now().UTC()
		for i := range 3 {
			seedProduct(t, pool, domain.Product{ID: fmt.Sprintf("lp-cur-%d", i), Name: fmt.Sprintf("Product %d", i), PriceCents: 1000, CategoryID: cat}, now.Add(time.Duration(i)*time.Second))
		}

		rec1 := httptest.NewRecorder()
		req1 := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id="+cat+"&page_size=2", nil)
		mux.ServeHTTP(rec1, req1)
		require.Equal(t, http.StatusOK, rec1.Code)

		var env1 struct {
			Meta struct {
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &env1))

		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?category_id="+cat+"&page_size=2&cursor="+env1.Meta.NextCursor, nil)
		mux.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)

		var env2 struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
			Meta struct {
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &env2))
		require.Len(t, env2.Data, 1)
		assert.False(t, env2.Meta.HasMore)
		assert.Empty(t, env2.Meta.NextCursor)
	})

	t.Run("Invalid page_size returns 400", func(t *testing.T) {
		for _, q := range []string{"page_size=0", "page_size=101"} {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?"+q, nil)
			mux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code, q)
		}
	})

	t.Run("Non-integer page_size returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?page_size=xyz", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Invalid cursor returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products?cursor=not-a-cursor", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
