package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMethod(t *testing.T) {
	t.Run("Blank name is rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		req := validCreateMethodRequestFeature()
		req.Name = ""
		_, err := svc.CreateMethod(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Invalid day estimates are rejected", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		req := validCreateMethodRequestFeature()
		req.EstimatedMinDays = 5
		req.EstimatedMaxDays = 3
		_, err := svc.CreateMethod(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransition)
	})

	t.Run("Method got created with all fields", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)

		created, err := svc.CreateMethod(t.Context(), CreateMethodRequest{Name: "Standard", Description: "Standard delivery", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
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
