//go:build integration

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMethodsIntegration(t *testing.T) {
	t.Run("Active methods are returned", func(t *testing.T) {
		svc, _, _, _, _ := newFeatureServices(t)
		mustCreateMethod(t, svc, "Standard", true)
		methods, err := svc.ListMethods(t.Context())
		require.NoError(t, err)
		count := 0
		for _, m := range methods {
			if m.Active {
				count++
			}
		}
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("Inactive methods are excluded", func(t *testing.T) {
		svc, _, _, _, _ := newFeatureServices(t)
		active := mustCreateMethod(t, svc, "Standard", true)
		inactive := mustCreateMethod(t, svc, "Express", false)
		methods, err := svc.ListMethods(t.Context())
		require.NoError(t, err)
		foundActive, foundInactive := false, false
		for _, m := range methods {
			if m.ID == active.ID {
				foundActive = true
			}
			if m.ID == inactive.ID {
				foundInactive = true
			}
		}
		assert.True(t, foundActive)
		assert.False(t, foundInactive)
	})
}
