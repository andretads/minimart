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

// updateFailingRepo serves an existing product but fails to persist the update,
// so the application error path can be exercised.
type updateFailingRepo struct {
	catalogRepoStub
	product domain.Product
	err     error
}

func (f updateFailingRepo) GetByID(context.Context, string) (domain.Product, error) {
	return f.product, nil
}

func (f updateFailingRepo) Update(context.Context, domain.Product) (domain.Product, error) {
	return domain.Product{}, f.err
}

func TestUpdateProduct(t *testing.T) {
	tests := []struct {
		name      string
		req       UpdateProductRequest
		wantErr   error
		wantName  string
		wantPrice int64
		wantStock int
	}{
		{"valid", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: 1, CategoryID: "cat-1"}, nil, "Gadget", 750, 3},
		{"trims surrounding space", UpdateProductRequest{ID: "p1", Name: "  Gadget  ", PriceCents: 750, Version: 1, CategoryID: "cat-1"}, nil, "Gadget", 750, 3},
		{"empty name", UpdateProductRequest{ID: "p1", Name: "   ", PriceCents: 750, Version: 1, CategoryID: "cat-1"}, domain.ErrEmptyName, "", 0, 0},
		{"zero price", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 0, Version: 1, CategoryID: "cat-1"}, domain.ErrNonPositivePrice, "", 0, 0},
		{"negative price", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: -1, Version: 1, CategoryID: "cat-1"}, domain.ErrNonPositivePrice, "", 0, 0},
		{"missing id", UpdateProductRequest{ID: "missing", Name: "Gadget", PriceCents: 750, Version: 1, CategoryID: "cat-1"}, domain.ErrProductNotFound, "", 0, 0},
		{"missing version", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, CategoryID: "cat-1"}, domain.ErrInvalidVersion, "", 0, 0},
		{"negative version", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: -1, CategoryID: "cat-1"}, domain.ErrInvalidVersion, "", 0, 0},
		{"stale version", UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: 99, CategoryID: "cat-1"}, domain.ErrProductOptimisticLock, "", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := memory.New()
			existing, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
			require.NoError(t, err)
			require.NoError(t, repo.Save(t.Context(), existing))

			svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
			got, err := svc.UpdateProduct(t.Context(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "p1", got.ID)
			assert.Equal(t, tt.wantName, got.Name)
			assert.Equal(t, tt.wantPrice, got.PriceCents)
			assert.Equal(t, tt.wantStock, got.Stock, "stock must be preserved")

			// The change is durable in the store, not just the returned value.
			stored, err := repo.GetByID(t.Context(), "p1")
			require.NoError(t, err)
			assert.Equal(t, got, stored)
		})
	}
}

func TestUpdateProductPersistenceFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	existing, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)

	m := &countingMetrics{}
	svc := NewService(updateFailingRepo{product: existing, err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err = svc.UpdateProduct(t.Context(), UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: 1, CategoryID: "cat-1"})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.updated)
}

// TestUpdateProductOptimisticLock proves a stale-version write surfaces the
// domain conflict sentinel and is counted as an error.
func TestUpdateProductOptimisticLock(t *testing.T) {
	existing, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)

	m := &countingMetrics{}
	svc := NewService(updateFailingRepo{product: existing, err: domain.ErrProductOptimisticLock}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err = svc.UpdateProduct(t.Context(), UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: 1, CategoryID: "cat-1"})
	require.ErrorIs(t, err, domain.ErrProductOptimisticLock)
	assert.Equal(t, []string{metrics.StatusError}, m.updated)
}

func TestUpdateProductRecordsSuccess(t *testing.T) {
	repo := memory.New()
	existing, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), existing))

	m := &countingMetrics{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err = svc.UpdateProduct(t.Context(), UpdateProductRequest{ID: "p1", Name: "Gadget", PriceCents: 750, Version: 1, CategoryID: "cat-1"})
	require.NoError(t, err)
	assert.Equal(t, []string{metrics.StatusSuccess}, m.updated)
}
