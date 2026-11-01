package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMethod(t *testing.T) {
	t.Run("Method does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.UpdateMethod(t.Context(), "missing", UpdateMethodRequest{Name: "Standard", Description: "desc", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodNotFound)
	})

	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		m := createTestMethodFeature(t, svc, "Standard", true)

		_, err := svc.UpdateMethod(t.Context(), m.ID, UpdateMethodRequest{Name: "", Description: "desc", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Invalid day estimates are rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		m := createTestMethodFeature(t, svc, "Standard", true)

		_, err := svc.UpdateMethod(t.Context(), m.ID, UpdateMethodRequest{Name: "Standard", Description: "desc", Active: true, EstimatedMinDays: 5, EstimatedMaxDays: 3})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Method got updated", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		m := createTestMethodFeature(t, svc, "Standard", true)

		_, err := svc.UpdateMethod(t.Context(), m.ID, UpdateMethodRequest{Name: "Express", Description: "Express delivery", Active: false, EstimatedMinDays: 1, EstimatedMaxDays: 2})
		require.NoError(t, err)

		stored, err := repo.GetMethod(t.Context(), m.ID)
		require.NoError(t, err)
		assert.Equal(t, "Express", stored.Name)
		assert.Equal(t, "Express delivery", stored.Description)
		assert.False(t, stored.Active)
		assert.Equal(t, 1, stored.EstimatedMinDays)
		assert.Equal(t, 2, stored.EstimatedMaxDays)
	})
}
