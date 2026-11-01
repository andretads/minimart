package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRate(t *testing.T) {
	t.Run("Rate does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.GetRate(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateNotFound)
	})

	t.Run("The rate is returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")
		m := createTestMethodFeature(t, svc, "Standard", true)
		r := createTestRateFeature(t, svc, z.ID, m.ID, 599, 5000, true)

		got, err := svc.GetRate(t.Context(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, r.ID, got.ID)
		assert.Equal(t, int64(599), got.BaseCostCents)
	})
}
