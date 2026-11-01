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

	showcasehttp "minimart/internal/showcase/presentation/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingReviewRepo struct{}

func (failingReviewRepo) List(ctx context.Context, limit int, after *domain.Cursor) ([]domain.Product, error) {
	return nil, nil
}

func (failingReviewRepo) ListByCategory(ctx context.Context, categoryID string, limit int, after *domain.Cursor) ([]domain.Product, error) {
	return nil, nil
}

func (failingReviewRepo) GetByID(ctx context.Context, id string) (domain.Product, error) {
	return domain.Product{}, nil
}

func (failingReviewRepo) ListImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	return nil, nil
}

func (failingReviewRepo) ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	return map[string][]domain.ProductImage{}, nil
}

func (failingReviewRepo) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	return nil, errRepoFailure
}

func (failingReviewRepo) CountReviews(ctx context.Context, productID string) (int, error) {
	return 0, errRepoFailure
}

func (failingReviewRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}

var errRepoFailure = &repoFailureError{}

type repoFailureError struct{}

func (repoFailureError) Error() string { return "db error" }

func TestGetProductReviewsIntegration(t *testing.T) {
	readPool, _ := mustPools(t)
	repo := postgres.NewRepository(readPool)
	svc := application.NewService(repo, stubSigner{}, time.Hour)
	handlers := showcasehttp.NewHandlers(svc)
	mux := http.NewServeMux()
	handlers.Register(mux)
	pool := readPool

	t.Run("Product with approved reviews returns them newest first", func(t *testing.T) {
		seedCategory(t, pool, "cat-shared", "Shared")
		seedProduct(t, pool, domain.Product{ID: "pr-1", Name: "Product A", PriceCents: 1000, CategoryID: "cat-shared"}, time.Now().UTC())
		now := time.Now().UTC()
		seedReview(t, pool, domain.Review{ID: "", ProductID: "pr-1", CustomerID: "c1", Description: "Great", Rating: 5, Status: "approved", CreatedAt: now.Add(-time.Hour)})
		seedReview(t, pool, domain.Review{ID: "", ProductID: "pr-1", CustomerID: "c2", Description: "Good", Rating: 3, Status: "approved", CreatedAt: now})

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/pr-1/reviews", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				ID          string `json:"id"`
				ProductID   string `json:"product_id"`
				CustomerID  string `json:"customer_id"`
				Description string `json:"description"`
				Rating      int    `json:"rating"`
				Status      string `json:"status"`
				CreatedAt   string `json:"created_at"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		require.Len(t, env.Data, 2)
		assert.Equal(t, "c2", env.Data[0].CustomerID)
		assert.Equal(t, "c1", env.Data[1].CustomerID)
		for _, r := range env.Data {
			assert.NotEmpty(t, r.ID)
			assert.Equal(t, "pr-1", r.ProductID)
			assert.NotEmpty(t, r.CustomerID)
			assert.NotEmpty(t, r.Description)
			assert.NotZero(t, r.Rating)
			assert.Equal(t, "approved", r.Status)
			assert.NotEmpty(t, r.CreatedAt)
		}
	})

	t.Run("Pending reviews are excluded", func(t *testing.T) {
		seedCategory(t, pool, "cat-shared", "Shared")
		seedProduct(t, pool, domain.Product{ID: "pr-2", Name: "Product B", PriceCents: 1000, CategoryID: "cat-shared"}, time.Now().UTC())
		now := time.Now().UTC()
		seedReview(t, pool, domain.Review{ID: "", ProductID: "pr-2", CustomerID: "c1", Description: "Approved", Rating: 5, Status: "approved", CreatedAt: now})
		seedReview(t, pool, domain.Review{ID: "", ProductID: "pr-2", CustomerID: "c2", Description: "Pending", Rating: 3, Status: "pending", CreatedAt: now})

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/pr-2/reviews", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    []struct {
				CustomerID string `json:"customer_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data, 1)
		assert.Equal(t, "c1", env.Data[0].CustomerID)
	})

	t.Run("Product without reviews returns an empty array", func(t *testing.T) {
		seedCategory(t, pool, "cat-shared", "Shared")
		seedProduct(t, pool, domain.Product{ID: "pr-3", Name: "Product C", PriceCents: 1000, CategoryID: "cat-shared"}, time.Now().UTC())

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/pr-3/reviews", nil)
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool  `json:"success"`
			Data    []any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.True(t, env.Success)
		assert.Empty(t, env.Data)
	})

	t.Run("Repository failure returns 500", func(t *testing.T) {
		failSvc := application.NewService(failingReviewRepo{}, stubSigner{}, time.Hour)
		failHandlers := showcasehttp.NewHandlers(failSvc)
		failMux := http.NewServeMux()
		failHandlers.Register(failMux)

		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "GET", "/v1/showcase/products/p1/reviews", nil)
		failMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.False(t, env.Success)
		assert.Equal(t, "erro interno", env.Error.Message)
	})
}

var _ = pgxpool.New
