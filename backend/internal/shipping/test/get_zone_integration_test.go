//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetZoneIntegration(t *testing.T) {
	shipSvc, _, _, _, _ := newFeatureServices(t)

	t.Run("Zone does not exist", func(t *testing.T) {
		_, err := shipSvc.GetZone(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("The zone is returned", func(t *testing.T) {
		z := mustCreateZone(t, shipSvc, "Brazil", "BR")
		got, err := shipSvc.GetZone(t.Context(), z.ID)
		require.NoError(t, err)
		assert.Equal(t, z.ID, got.ID)
		assert.Equal(t, "BR", got.Country)
	})
}
