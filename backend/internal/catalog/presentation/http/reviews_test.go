package http

import (
	"encoding/json"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddReview(t *testing.T) {
	mux, repo, customer := newTestMuxWithCustomer(t)

	// Seed a product
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	t.Run("adds a review successfully", func(t *testing.T) {
		body := `{"description":"Great product!","rating":5}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		require.Equal(t, http.StatusCreated, rec.Code)

		var env struct {
			Data reviewResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "p1", env.Data.ProductID)
		assert.Equal(t, "cust-1", env.Data.CustomerID)
		assert.Equal(t, "Great product!", env.Data.Description)
		assert.Equal(t, 5, env.Data.Rating)
		assert.Equal(t, "pending", env.Data.Status)
		assert.NotEmpty(t, env.Data.ID)
	})

	t.Run("attributes the review to the session customer", func(t *testing.T) {
		*repo = *memory.New()
		p, _ := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		_ = repo.Save(t.Context(), p)
		body := `{"description":"Great product!","rating":5}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-42")
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "user-42", customer.gotUser)
	})

	t.Run("ignores a customer id sent in the body", func(t *testing.T) {
		*repo = *memory.New()
		p, _ := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		_ = repo.Save(t.Context(), p)
		body := `{"customer_id":"someone-else","description":"Great product!","rating":5}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		require.Equal(t, http.StatusCreated, rec.Code)

		var env struct {
			Data reviewResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "cust-1", env.Data.CustomerID)
	})

	t.Run("rejects an unauthenticated request", func(t *testing.T) {
		body := `{"description":"Great product!","rating":5}`
		rec := do(t, mux, http.MethodPost, "/v1/products/p1/reviews", body)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects invalid rating too low", func(t *testing.T) {
		body := `{"description":"Bad","rating":0}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects invalid rating too high", func(t *testing.T) {
		body := `{"description":"Bad","rating":6}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects empty description", func(t *testing.T) {
		body := `{"description":"   ","rating":5}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 404 for missing product", func(t *testing.T) {
		body := `{"description":"Good","rating":4}`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/missing/reviews", body, "user-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		body := `{`
		rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAddReview_CustomerResolutionFailure(t *testing.T) {
	mux, _, customer := newTestMuxWithCustomer(t)
	customer.err = assert.AnError

	body := `{"description":"Great product!","rating":5}`
	rec := doAuthenticated(t, mux, http.MethodPost, "/v1/products/p1/reviews", body, "user-1")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetReviews(t *testing.T) {
	mux, repo := newTestMux(t)

	// Seed a product
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	// Add some reviews directly to the repo
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Bad", Rating: 2, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r3", ProductID: "p1", CustomerID: "c3", Description: "Pending", Rating: 3, Status: "pending", CreatedAt: time.Now().UTC()}))

	t.Run("returns approved reviews", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/products/p1/reviews", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data []reviewResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Len(t, env.Data, 2)
		assert.Equal(t, "r2", env.Data[0].ID) // newest first
		assert.Equal(t, "r1", env.Data[1].ID)
	})

	t.Run("returns empty list for missing product (no error)", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/products/missing/reviews", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data []reviewResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Empty(t, env.Data)
	})
}
