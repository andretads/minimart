//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRateIntegration(t *testing.T) {
	svc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Rate does not exist", func(t *testing.T) {
		_, err := svc.UpdateRate(t.Context(), "missing", application.UpdateRateRequest{BaseCostCents: 299, FreeThresholdCents: 0, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRateNotFound)
	})

	t.Run("Negative base cost is rejected", func(t *testing.T) {
		z := mustCreateZone(t, svc, "Brazil", "BR")
		m := mustCreateMethod(t, svc, "Standard", true)
		r := mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.UpdateRate(t.Context(), r.ID, application.UpdateRateRequest{BaseCostCents: -1, FreeThresholdCents: 5000, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Negative free threshold is rejected", func(t *testing.T) {
		z := mustCreateZone(t, svc, "Brazil", "BR")
		m := mustCreateMethod(t, svc, "Standard", true)
		r := mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.UpdateRate(t.Context(), r.ID, application.UpdateRateRequest{BaseCostCents: 599, FreeThresholdCents: -1, Active: true})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Rate got updated", func(t *testing.T) {
		z := mustCreateZone(t, svc, "Brazil", "BR")
		m := mustCreateMethod(t, svc, "Standard", true)
		r := mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		_, err := svc.UpdateRate(t.Context(), r.ID, application.UpdateRateRequest{BaseCostCents: 299, FreeThresholdCents: 0, Active: false})
		require.NoError(t, err)
		stored, err := repo.GetRate(t.Context(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(299), stored.BaseCostCents)
		assert.Equal(t, int64(0), stored.FreeThresholdCents)
		assert.False(t, stored.Active)
	})
}
