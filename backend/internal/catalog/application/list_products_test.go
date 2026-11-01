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

// listProductsErrRepo makes List fail.
type listProductsErrRepo struct {
	catalogRepoStub
	err error
}

func (f listProductsErrRepo) List(context.Context) ([]domain.Product, error) {
	return nil, f.err
}

func TestListProducts(t *testing.T) {
	t.Run("returns every stored product", func(t *testing.T) {
		repo := memory.New()
		seedProduct(t, repo, "p1", 1)
		seedProduct(t, repo, "p2", 2)
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		got, err := svc.ListProducts(t.Context())
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("passes repository errors through unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		svc := NewService(listProductsErrRepo{err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		_, err := svc.ListProducts(t.Context())
		assert.ErrorIs(t, err, boom)
	})
}
