//go:build integration

package test

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMethodIntegration(t *testing.T) {
	shipSvc, _, _, _, _ := newFeatureServices(t)

	t.Run("Method does not exist", func(t *testing.T) {
		_, err := shipSvc.GetMethod(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodNotFound)
	})

	t.Run("The method is returned", func(t *testing.T) {
		m, err := shipSvc.CreateMethod(t.Context(), application.CreateMethodRequest{Name: "Standard-" + uniqueShippingID(), Description: "Standard delivery", Active: true, EstimatedMinDays: 3, EstimatedMaxDays: 5})
		require.NoError(t, err)
		got, err := shipSvc.GetMethod(t.Context(), m.ID)
		require.NoError(t, err)
		assert.Equal(t, m.ID, got.ID)
		assert.Equal(t, m.Name, got.Name)
	})
}
