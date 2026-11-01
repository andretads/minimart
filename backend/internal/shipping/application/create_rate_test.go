package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRate(t *testing.T) {
	t.Run("Blank zone id is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: "", MethodID: "sm1", BaseCostCents: 599, FreeThresholdCents: 5000, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateNotFound)
	})

	t.Run("Blank method id is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: "sz1", MethodID: "", BaseCostCents: 599, FreeThresholdCents: 5000, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateNotFound)
	})

	t.Run("Negative base cost is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: "sz1", MethodID: "sm1", BaseCostCents: -1, FreeThresholdCents: 5000, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Negative free threshold is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: "sz1", MethodID: "sm1", BaseCostCents: 599, FreeThresholdCents: -1, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Rate got created with all fields", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		z := createTestZoneFeature(t, svc, "Brazil", "BR")
		m := createTestMethodFeature(t, svc, "Standard", true)

		created, err := svc.CreateRate(t.Context(), CreateRateRequest{ZoneID: z.ID, MethodID: m.ID, BaseCostCents: 599, FreeThresholdCents: 5000, Active: true})
		require.NoError(t, err)

		stored, err := repo.GetRate(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, z.ID, stored.ZoneID)
		assert.Equal(t, m.ID, stored.MethodID)
		assert.Equal(t, int64(599), stored.BaseCostCents)
		assert.Equal(t, int64(5000), stored.FreeThresholdCents)
		assert.True(t, stored.Active)
		assert.False(t, stored.CreatedAt.IsZero())
	})
}
