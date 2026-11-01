package application

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getProductErrRepo makes GetByID fail with a non-not-found error.
type getProductErrRepo struct {
	catalogRepoStub
	err error
}

func (f getProductErrRepo) GetByID(context.Context, string) (domain.Product, error) {
	return domain.Product{}, f.err
}

func TestGetProduct(t *testing.T) {
	t.Run("returns the stored product", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 5)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, 5, got.Stock)
	})

	t.Run("missing product reports not found", func(t *testing.T) {
		svc := NewService(memory.New(), fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.GetProduct(t.Context(), "missing")
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("passes repository errors through unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		svc := NewService(getProductErrRepo{err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.GetProduct(t.Context(), "p1")
		assert.ErrorIs(t, err, boom)
	})
}
