//go:build integration

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListZonesIntegration(t *testing.T) {
	t.Run("Zones are returned", func(t *testing.T) {
		svc, _, _, _, _ := newFeatureServices(t)
		mustCreateZone(t, svc, "Brazil", "BR")
		zones, err := svc.ListZones(t.Context())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(zones), 1)
	})
}
