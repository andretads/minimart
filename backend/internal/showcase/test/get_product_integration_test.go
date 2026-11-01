//go:build integration

package test

import (
	"encoding/json"
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/postgres"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	showcasehttp "minimart/internal/showcase/presentation/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProductIntegration(t *testing.T) {
	readPool, _ := mustPools(t)
	repo := postgres.NewRepository(readPool)
	svc := application.NewService(repo, stubSigner{}, time.Hour)
	handlers := showcasehttp.NewHandlers(svc)
	mux := http.NewServeMux()
	handlers.Register(mux)
	pool := readPool

	t.Run("Get product by ID returns product details", func(t *testing.T) {
		seedCategory(t, pool, "cat-shared", "Shared")
		seedProduct(t, pool, domain.Product{
			ID: "gp-1", Name: "Product A", PriceCents: 1000, Stock: 5, SKU: "SKU-1", Description: "A fine product", CategoryID: "cat-shared",
		}, time.Now().UTC())

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/gp-1", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				PriceCents   int64  `json:"price_cents"`
				Stock        int    `json:"stock"`
				SKU          string `json:"sku"`
				Description  string `json:"description"`
				Details      []any  `json:"details"`
				TotalRating  int64  `json:"total_rating"`
				TotalReviews int64  `json:"total_reviews"`
				Images       []any  `json:"images"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		assert.Equal(t, "gp-1", env.Data.ID)
		assert.Equal(t, "Product A", env.Data.Name)
		assert.Equal(t, int64(1000), env.Data.PriceCents)
		assert.Equal(t, 5, env.Data.Stock)
		assert.Equal(t, "SKU-1", env.Data.SKU)
		assert.Equal(t, "A fine product", env.Data.Description)
		assert.Equal(t, int64(0), env.Data.TotalRating)
		assert.Equal(t, int64(0), env.Data.TotalReviews)
	})

	t.Run("Get product by ID includes images with presigned URLs", func(t *testing.T) {
		seedCategory(t, pool, "cat-shared", "Shared")
		seedProduct(t, pool, domain.Product{ID: "gp-2", Name: "Product B", PriceCents: 2000, CategoryID: "cat-shared"}, time.Now().UTC())
		seedImage(t, pool, domain.ProductImage{ID: "img-1", ProductID: "gp-2", ObjectKey: "key-1", Position: 1})
		seedImage(t, pool, domain.ProductImage{ID: "img-2", ProductID: "gp-2", ObjectKey: "key-2", Position: 2})

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/gp-2", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    struct {
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
				ImageURL string `json:"image_url"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data.Images, 2)
		for _, img := range env.Data.Images {
			assert.NotEmpty(t, img.URL)
		}
		assert.NotEmpty(t, env.Data.ImageURL)
		assert.Equal(t, env.Data.ImageURL, env.Data.Images[0].URL)
	})

	t.Run("Get product not found returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/nonexistent", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.False(t, env.Success)
		assert.Equal(t, "produto não encontrado", env.Error.Message)
	})
}

var _ = pgxpool.New
