//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMethodIntegration(t *testing.T) {
	shipSvc, _, _, repo, _ := newFeatureServices(t)

	t.Run("Blank name is rejected", func(t *testing.T) {
		_, err := shipSvc.CreateMethod(t.Context(), application.CreateMethodRequest{Name: "", Description: "desc", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Invalid day estimates are rejected", func(t *testing.T) {
		_, err := shipSvc.CreateMethod(t.Context(), application.CreateMethodRequest{Name: "Standard-" + uniqueShippingID(), Description: "desc", Active: true, EstimatedMinDays: 5, EstimatedMaxDays: 3})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Method got created with all fields", func(t *testing.T) {
		created, err := shipSvc.CreateMethod(t.Context(), application.CreateMethodRequest{Name: "Standard", Description: "Standard delivery", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
		require.NoError(t, err)
		stored, err := repo.GetMethod(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "Standard", stored.Name)
		assert.Equal(t, "Standard delivery", stored.Description)
		assert.True(t, stored.Active)
		assert.Equal(t, 3, stored.EstimatedMinDays)
		assert.Equal(t, 5, stored.EstimatedMaxDays)
		assert.False(t, stored.CreatedAt.IsZero())
	})
}
