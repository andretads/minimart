package http

import (
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetProductIncludesDetailsAndImages(t *testing.T) {
	mux, repo := newTestMux(t)

	seedProduct(t, repo, domain.Product{
		ID:           "p1",
		Name:         "Product 1",
		PriceCents:   1999,
		CategoryID:   "cat-1",
		Stock:        7,
		SKU:          "SKU-1",
		Description:  "A fine product",
		Details:      []domain.ProductDetail{{Name: "Color", Value: "Red"}},
		TotalRating:  18,
		TotalReviews: 4,
	})
	seedImage(t, repo, domain.ProductImage{ID: "img-2", ProductID: "p1", ObjectKey: "products/p1/b.jpg", ContentType: "image/jpeg", SizeBytes: 200, Position: 1})
	seedImage(t, repo, domain.ProductImage{ID: "img-1", ProductID: "p1", ObjectKey: "products/p1/a.jpg", ContentType: "image/jpeg", SizeBytes: 100, Position: 0})

	rec := do(t, mux, "GET", "/v1/showcase/products/p1")
	env := assertJSONResponse(t, rec, 200, true)

	var p productResponse
	unmarshalData(t, env, &p)

	require.Equal(t, 7, p.Stock)
	require.Equal(t, "SKU-1", p.SKU)
	require.Equal(t, "A fine product", p.Description)
	require.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Red"}}, p.Details)
	require.Equal(t, int64(18), p.TotalRating)
	require.Equal(t, int64(4), p.TotalReviews)
	require.Equal(t, "https://cdn.test/products/p1/a.jpg", p.ImageURL)
	require.Len(t, p.Images, 2)
	require.Equal(t, "img-1", p.Images[0].ID)
	require.Equal(t, "https://cdn.test/products/p1/a.jpg", p.Images[0].URL)
	require.Equal(t, 0, p.Images[0].Position)
	require.Equal(t, "img-2", p.Images[1].ID)
}

func TestListProductsIncludesSummaryFieldsAndPrimaryImage(t *testing.T) {
	mux, repo := newTestMux(t)

	seedProduct(t, repo, domain.Product{
		ID:           "p1",
		Name:         "Product 1",
		PriceCents:   1999,
		Stock:        3,
		TotalRating:  9,
		TotalReviews: 2,
	})
	seedImage(t, repo, domain.ProductImage{ID: "img-1", ProductID: "p1", ObjectKey: "products/p1/a.jpg", ContentType: "image/jpeg", SizeBytes: 100, Position: 0})

	rec := do(t, mux, "GET", "/v1/showcase/products")
	env := assertJSONResponse(t, rec, 200, true)

	var items []productResponse
	unmarshalData(t, env, &items)
	require.Len(t, items, 1)
	require.Equal(t, 3, items[0].Stock)
	require.Equal(t, int64(9), items[0].TotalRating)
	require.Equal(t, int64(2), items[0].TotalReviews)
	require.Equal(t, "https://cdn.test/products/p1/a.jpg", items[0].ImageURL)
	require.Empty(t, items[0].Images)
}

func TestListProductReviews(t *testing.T) {
	mux, repo := newTestMux(t)

	now := time.Now().UTC().Truncate(time.Second)
	seedReview(t, repo, domain.Review{
		ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Great", Rating: 5, Status: "approved", CreatedAt: now,
	})
	seedReview(t, repo, domain.Review{
		ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Pending", Rating: 3, Status: "pending", CreatedAt: now,
	})

	rec := do(t, mux, "GET", "/v1/showcase/products/p1/reviews")
	env := assertJSONResponse(t, rec, 200, true)

	var reviews []reviewResponse
	unmarshalData(t, env, &reviews)
	require.Len(t, reviews, 1)
	require.Equal(t, "r1", reviews[0].ID)
	require.Equal(t, "c1", reviews[0].CustomerID)
	require.Equal(t, 5, reviews[0].Rating)
	require.Equal(t, "approved", reviews[0].Status)
	require.True(t, now.Equal(reviews[0].CreatedAt))
}

func TestListProductReviewsEmpty(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := do(t, mux, "GET", "/v1/showcase/products/p1/reviews")
	env := assertJSONResponse(t, rec, 200, true)

	var reviews []reviewResponse
	unmarshalData(t, env, &reviews)
	require.Empty(t, reviews)
}
