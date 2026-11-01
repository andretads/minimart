//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateZoneIntegration(t *testing.T) {
	shipSvc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Blank name is rejected", func(t *testing.T) {
		_, err := shipSvc.CreateZone(t.Context(), application.CreateZoneRequest{Name: "", Country: "BR"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("Blank country is rejected", func(t *testing.T) {
		_, err := shipSvc.CreateZone(t.Context(), application.CreateZoneRequest{Name: "Brazil", Country: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrZoneNotFound)
	})

	t.Run("Zone got created with all fields", func(t *testing.T) {
		created, err := shipSvc.CreateZone(t.Context(), application.CreateZoneRequest{Name: "Brazil", Country: "BR", State: "SP", City: "Sao Paulo", PostalCodeStart: "01000-000", PostalCodeEnd: "09999-999"})
		require.NoError(t, err)
		stored, err := repo.GetZone(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Brazil", stored.Name)
		assert.Equal(t, "BR", stored.Country)
		assert.Equal(t, "SP", stored.State)
		assert.Equal(t, "Sao Paulo", stored.City)
		assert.Equal(t, "01000-000", stored.PostalCodeStart)
		assert.Equal(t, "09999-999", stored.PostalCodeEnd)
		assert.False(t, stored.CreatedAt.IsZero())
	})
}
