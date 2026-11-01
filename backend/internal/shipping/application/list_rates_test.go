package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRates(t *testing.T) {
	t.Run("Rates are returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")
		m := createTestMethodFeature(t, svc, "Standard", true)
		createTestRateFeature(t, svc, z.ID, m.ID, 599, 5000, true)

		rates, err := svc.ListRates(t.Context(), z.ID, m.ID)
		require.NoError(t, err)
		assert.Len(t, rates, 1)
	})
}
