package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateZone(t *testing.T) {
	t.Run("Zone does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.UpdateZone(t.Context(), "missing", UpdateZoneRequest{Name: "Brazil", Country: "BR"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")

		_, err := svc.UpdateZone(t.Context(), z.ID, UpdateZoneRequest{Name: "", Country: "BR"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("Blank country is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")

		_, err := svc.UpdateZone(t.Context(), z.ID, UpdateZoneRequest{Name: "Brazil", Country: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("Zone got updated", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")

		_, err := svc.UpdateZone(t.Context(), z.ID, UpdateZoneRequest{Name: "Brazil", Country: "BR", State: "RJ", City: "Rio", PostalCodeStart: "20000-000", PostalCodeEnd: "29999-999"})
		require.NoError(t, err)

		stored, err := repo.GetZone(t.Context(), z.ID)
		require.NoError(t, err)
		assert.Equal(t, "Brazil", stored.Name)
		assert.Equal(t, "BR", stored.Country)
		assert.Equal(t, "RJ", stored.State)
		assert.Equal(t, "Rio", stored.City)
		assert.Equal(t, "20000-000", stored.PostalCodeStart)
		assert.Equal(t, "29999-999", stored.PostalCodeEnd)
	})
}
