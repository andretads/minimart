//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRateIntegration(t *testing.T) {
	shipSvc, _, _, _, _ := newFeatureServices(t)

	t.Run("Rate does not exist", func(t *testing.T) {
		_, err := shipSvc.GetRate(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateNotFound)
	})

	t.Run("The rate is returned", func(t *testing.T) {
		z := mustCreateZone(t, shipSvc, "Brazil", "BR")
		m := mustCreateMethod(t, shipSvc, "Standard", true)
		r := mustCreateRate(t, shipSvc, z.ID, m.ID, 599, 5000, true)
		got, err := shipSvc.GetRate(t.Context(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
		assert.Equal(t, int64(599), got.BaseCostCents)
	})
}
