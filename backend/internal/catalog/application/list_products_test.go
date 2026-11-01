package application

import (
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	t.Run("Empty catalog returns an empty list", func(t *testing.T) {
		svc, _ := newTestService(t)

		got, err := svc.ListProducts(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("All stored products are returned", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 1)
		seedProduct(t, repo, "p2", 1)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.ListProducts(t.Context())
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}
