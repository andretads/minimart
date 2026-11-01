//go:build integration

package test

import (
	"minimart/internal/catalog/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReviewSummaryIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		_, _, _, err := env.Svc.GetReviewSummary(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Review summary got returned", func(t *testing.T) {
		seedCatalogProductFull(t, env, "p1", "Widget", 500, 3, "", "", nil)
		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		p.TotalRating = 20
		p.TotalReviews = 5
		require.NoError(t, env.Repo.Save(t.Context(), p))

		totalRating, totalReviews, avg, err := env.Svc.GetReviewSummary(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, int64(20), totalRating)
		assert.Equal(t, int64(5), totalReviews)
		assert.InDelta(t, 4.0, avg, 0.0001)
	})
}
