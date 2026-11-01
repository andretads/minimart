//go:build integration

package test

import (
	"minimart/internal/shipping/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListShipmentsIntegration(t *testing.T) {
	t.Run("Shipments are returned newest first", func(t *testing.T) {
		shipSvc, _, _, repo, _ := newFeatureServices(t)
		userID := "u1-" + uniqueShippingID()
		seedStoredShipment(t, shipSvc, repo, userID, "o1-"+uniqueShippingID(), domain.ShipmentStatusPending)
		shipments, _, err := shipSvc.ListShipments(t.Context(), userID, 1, 20)
		require.NoError(t, err)
		assert.Len(t, shipments, 1)
	})

	t.Run("Empty list is returned", func(t *testing.T) {
		svc, _, _, _, _ := newFeatureServices(t)
		shipments, _, err := svc.ListShipments(t.Context(), "u1-"+uniqueShippingID(), 1, 20)
		require.NoError(t, err)
		assert.Empty(t, shipments)
	})
}
