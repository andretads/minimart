package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMethod(t *testing.T) {
	t.Run("Method does not exist", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		_, err := svc.GetMethod(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMethodNotFound)
	})

	t.Run("The method is returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		m := createTestMethodFeature(t, svc, "Standard", true)

		got, err := svc.GetMethod(t.Context(), m.ID)
		require.NoError(t, err)
		assert.Equal(t, m.ID, got.ID)
		assert.Equal(t, "Standard", got.Name)
	})
}
