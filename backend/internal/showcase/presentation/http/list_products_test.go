package http

import (
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	mux, repo := newTestMux(t)

	now := time.Now()
	products := []domain.Product{
		{ID: "p1", Name: "Product 1", PriceCents: 100, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "p2", Name: "Product 2", PriceCents: 200, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "p3", Name: "Product 3", PriceCents: 300, CreatedAt: now},
	}
	for _, p := range products {
		seedProduct(t, repo, p)
	}

	// newest-first order expected: p3, p2, p1

	t.Run("default page size returns all when <= page_size", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products")
		env := assertJSONResponse(t, rec, 200, true)
		var items []productResponse
		unmarshalData(t, env, &items)
		require.Len(t, items, 3)
		require.Equal(t, 20, env.Meta.PageSize)
		require.False(t, *env.Meta.HasMore)
		require.Empty(t, env.Meta.NextCursor)
		require.Equal(t, "Product 3", items[0].Name)
	})

	t.Run("page_size limits and sets cursor", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products?page_size=2")
		env := assertJSONResponse(t, rec, 200, true)
		var items []productResponse
		unmarshalData(t, env, &items)
		require.Len(t, items, 2)
		require.Equal(t, 2, env.Meta.PageSize)
		require.True(t, *env.Meta.HasMore)
		require.NotEmpty(t, env.Meta.NextCursor)
		require.Equal(t, "Product 3", items[0].Name)
		require.Equal(t, "Product 2", items[1].Name)
	})

	t.Run("follow cursor returns next page", func(t *testing.T) {
		// get cursor from first page
		rec := do(t, mux, "GET", "/v1/showcase/products?page_size=2")
		env := assertJSONResponse(t, rec, 200, true)
		cursor := env.Meta.NextCursor
		rec2 := do(t, mux, "GET", "/v1/showcase/products?page_size=2&cursor="+cursor)
		env2 := assertJSONResponse(t, rec2, 200, true)
		var items []productResponse
		unmarshalData(t, env2, &items)
		require.Len(t, items, 1)
		require.False(t, *env2.Meta.HasMore)
		require.Empty(t, env2.Meta.NextCursor)
		require.Equal(t, "Product 1", items[0].Name)
	})

	t.Run("invalid page_size zero", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products?page_size=0")
		env := assertJSONResponse(t, rec, 400, false)
		require.NotNil(t, env.Error)
	})

	t.Run("invalid page_size over max", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products?page_size=101")
		assertJSONResponse(t, rec, 400, false)
	})

	t.Run("non-integer page_size", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products?page_size=xyz")
		assertJSONResponse(t, rec, 400, false)
	})

	t.Run("invalid cursor", func(t *testing.T) {
		rec := do(t, mux, "GET", "/v1/showcase/products?cursor=not-a-cursor")
		assertJSONResponse(t, rec, 400, false)
	})
}

func TestListProductsEmpty(t *testing.T) {
	mux, _ := newTestMux(t)
	rec := do(t, mux, "GET", "/v1/showcase/products")
	env := assertJSONResponse(t, rec, 200, true)

	var items []productResponse
	unmarshalData(t, env, &items)
	require.Empty(t, items)
	require.Equal(t, 20, env.Meta.PageSize)
	require.False(t, *env.Meta.HasMore)
}
