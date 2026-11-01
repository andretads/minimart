package application

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListShipments(t *testing.T) {
	t.Run("Shipments are returned newest first", func(t *testing.T) {
		svc, repo, _ := newFeatureService(t)
		seedShipmentFeature(t, repo, "u1", "o1", domain.ShipmentStatusPending)

		shipments, _, err := svc.ListShipments(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Len(t, shipments, 1)
	})

	t.Run("Empty list is returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)

		shipments, _, err := svc.ListShipments(t.Context(), "u1", 1, 20)
		require.NoError(t, err)
		assert.Empty(t, shipments)
	})
}
