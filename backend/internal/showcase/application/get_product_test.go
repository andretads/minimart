package application

import (
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProduct(t *testing.T) {
	t.Run("Get product by ID returns product details", func(t *testing.T) {
		svc, repo := newTestService(t)

		now := time.Now().UTC()
		p := domain.Product{
			ID:           "p1",
			Name:         "Product A",
			PriceCents:   1000,
			Stock:        5,
			SKU:          "SKU-1",
			Description:  "A fine product",
			Details:      []domain.ProductDetail{{Name: "Color", Value: "Red"}},
			TotalRating:  0,
			TotalReviews: 0,
			CreatedAt:    now,
		}
		seedProduct(t, repo, p)

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, "Product A", got.Name)
		assert.Equal(t, int64(1000), got.PriceCents)
		assert.Equal(t, 5, got.Stock)
		assert.Equal(t, "SKU-1", got.SKU)
		assert.Equal(t, "A fine product", got.Description)
		assert.NotNil(t, got.Details)
		assert.Equal(t, int64(0), got.TotalRating)
		assert.Equal(t, int64(0), got.TotalReviews)
		assert.NotNil(t, got.Images)
	})

	t.Run("Get product by ID includes images with presigned URLs", func(t *testing.T) {
		svc, repo := newTestService(t)

		now := time.Now().UTC()
		seedProduct(t, repo, domain.Product{ID: "p1", Name: "Product A", PriceCents: 1000, CreatedAt: now})
		seedImage(t, repo, domain.ProductImage{ID: "i1", ProductID: "p1", ObjectKey: "key-1", Position: 1})
		seedImage(t, repo, domain.ProductImage{ID: "i2", ProductID: "p1", ObjectKey: "key-2", Position: 2})

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		require.Len(t, got.Images, 2)
		assert.Equal(t, 1, got.Images[0].Position)
		assert.Equal(t, 2, got.Images[1].Position)
	})

	t.Run("Get product not found returns 404", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetProduct(t.Context(), "nonexistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}
