//go:build integration

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRatesIntegration(t *testing.T) {
	t.Run("Rates are returned", func(t *testing.T) {
		svc, _, _, _, _ := newFeatureServices(t)
		z := mustCreateZone(t, svc, "Brazil-"+uniqueShippingID(), "BR")
		m := mustCreateMethod(t, svc, "Standard-"+uniqueShippingID(), true)
		mustCreateRate(t, svc, z.ID, m.ID, 599, 5000, true)
		rates, err := svc.ListRates(t.Context(), z.ID, m.ID)
		require.NoError(t, err)
		assert.Len(t, rates, 1)
	})
}
