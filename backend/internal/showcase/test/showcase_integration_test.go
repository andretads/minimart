//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/postgres"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	showcasehttp "minimart/internal/showcase/presentation/http"
)

func TestShowcaseIntegration(t *testing.T) {
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, testDSN)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.NewRepository(pool)
	svc := application.NewService(repo, nil, 0)
	handlers := showcasehttp.NewHandlers(svc)

	mux := http.NewServeMux()
	handlers.Register(mux)

	// Seed products with different created_at for ordering test
	seedProducts(t, ctx, pool)

	t.Run("list products default pagination", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products", nil)
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				PriceCents int64  `json:"price_cents"`
			} `json:"data"`
			Meta struct {
				PageSize   int    `json:"page_size"`
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		require.Len(t, env.Data, 5) // 5 seeded products
		require.Equal(t, 20, env.Meta.PageSize)
		require.False(t, env.Meta.HasMore)
		require.Empty(t, env.Meta.NextCursor)
	})

	t.Run("list products with page_size=2", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page_size=2", nil)
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				ID string `json:"id"`
			} `json:"data"`
			Meta struct {
				PageSize   int    `json:"page_size"`
				HasMore    bool   `json:"has_more"`
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		require.Len(t, env.Data, 2)
		require.Equal(t, 2, env.Meta.PageSize)
		require.True(t, env.Meta.HasMore)
		require.NotEmpty(t, env.Meta.NextCursor)
	})

	t.Run("pagination returns correct page", func(t *testing.T) {
		// Page 1
		rec1 := httptest.NewRecorder()
		req1 := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page=1&page_size=2", nil)
		mux.ServeHTTP(rec1, req1)
		require.Equal(t, http.StatusOK, rec1.Code)

		var env1 struct {
			Meta struct {
				NextCursor string `json:"next_cursor"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &env1))

		// Page 2
		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page_size=2&cursor="+env1.Meta.NextCursor, nil)
		mux.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)

		// Pages should have different items
		require.NotEqual(t, rec1.Body.String(), rec2.Body.String())
	})

	t.Run("get product by ID", func(t *testing.T) {
		// First, get a product ID from the list
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page_size=1", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var listResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
		require.Len(t, listResp.Data, 1)

		productID := listResp.Data[0].ID

		// Now fetch the product by ID
		rec2 := httptest.NewRecorder()
		req2 := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products/"+productID, nil)
		mux.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)

		var productResp struct {
			Success bool           `json:"success"`
			Data    domain.Product `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &productResp))
		require.True(t, productResp.Success)
		require.Equal(t, productID, productResp.Data.ID)
	})

	t.Run("get product not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products/nonexistent-id", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid page_size returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page_size=101", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid page_size zero returns 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/showcase/products?page_size=0", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func seedProducts(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	// First create a test category
	_, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, icon, version, created_at, updated_at)
		VALUES ('cat-1', 'Test Category', '', 1, NOW(), NOW())
	`)
	require.NoError(t, err)

	products := []domain.Product{
		{ID: "p1", Name: "Product A", PriceCents: 1000, CategoryID: "cat-1"},
		{ID: "p2", Name: "Product B", PriceCents: 2000, CategoryID: "cat-1"},
		{ID: "p3", Name: "Product C", PriceCents: 3000, CategoryID: "cat-1"},
		{ID: "p4", Name: "Product D", PriceCents: 4000, CategoryID: "cat-1"},
		{ID: "p5", Name: "Product E", PriceCents: 5000, CategoryID: "cat-1"},
	}

	// Insert with explicit created_at to control ordering (newest first)
	for i, p := range products {
		_, err := pool.Exec(ctx, `
			INSERT INTO products (id, name, price_cents, stock, version, created_at, updated_at, category_id)
			VALUES ($1, $2, $3, 10, 1, $4, $4, $5)
		`, p.ID, p.Name, p.PriceCents, time.Now().Add(time.Duration(i)*time.Second), p.CategoryID)
		require.NoError(t, err)
	}
}
