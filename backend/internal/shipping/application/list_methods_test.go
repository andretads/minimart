package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMethods(t *testing.T) {
	t.Run("Active methods are returned", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		createTestMethodFeature(t, svc, "Standard", true)

		methods, err := svc.ListMethods(t.Context())
		require.NoError(t, err)
		assert.Len(t, methods, 1)
	})

	t.Run("Inactive methods are excluded", func(t *testing.T) {
		svc, _, _ := newFeatureService(t)
		createTestMethodFeature(t, svc, "Standard", true)
		createTestMethodFeature(t, svc, "Express", false)

		methods, err := svc.ListMethods(t.Context())
		require.NoError(t, err)
		assert.Len(t, methods, 1)
	})
}
