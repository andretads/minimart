package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListZones(t *testing.T) {
	t.Run("Zones are returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		createTestZoneFeature(t, svc, "Brazil", "BR")

		zones, err := svc.ListZones(t.Context())
		require.NoError(t, err)
		assert.Len(t, zones, 1)
	})
}
