package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetZone(t *testing.T) {
	t.Run("Zone does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.GetZone(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("The zone is returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")

		got, err := svc.GetZone(t.Context(), z.ID)
		require.NoError(t, err)
		assert.Equal(t, z.ID, got.ID)
		assert.Equal(t, "BR", got.Country)
	})
}
