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

// failingRepo makes Save fail so the application error path can be exercised.
type failingRepo struct {
	catalogRepoStub
	err error
}

func (f failingRepo) Save(context.Context, domain.Product) error { return f.err }

// countingMetrics records product-creation results while no-op'ing the rest.
type countingMetrics struct {
	metrics.Noop
	created   []string
	updated   []string
	reserved  []string
	released  []string
	moderated []string
}

func (c *countingMetrics) IncProductCreated(status string) { c.created = append(c.created, status) }
func (c *countingMetrics) IncProductUpdated(status string) { c.updated = append(c.updated, status) }
func (c *countingMetrics) IncReviewAdded(status string)    { c.updated = append(c.updated, status) }
func (c *countingMetrics) IncReviewModerated(status string) {
	c.moderated = append(c.moderated, status)
}

func (c *countingMetrics) IncStockReserved(status string) { c.reserved = append(c.reserved, status) }

func (c *countingMetrics) IncStockReleased(status string) { c.released = append(c.released, status) }

// The test is itself a small composition root: it injects the in-memory adapter,
// no database required.
func TestCreateProduct(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateProductRequest
		wantErr error
	}{
		{"valid", CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"}, nil},
		{"empty name", CreateProductRequest{Name: "  ", PriceCents: 500, CategoryID: "cat-1"}, domain.ErrEmptyName},
		{"zero price", CreateProductRequest{Name: "Widget", PriceCents: 0, CategoryID: "cat-1"}, domain.ErrNonPositivePrice},
		{"negative stock", CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: -1, CategoryID: "cat-1"}, domain.ErrNegativeStock},
		{"missing category", CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: ""}, domain.ErrProductCategoryRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(memory.New(), fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
			got, err := svc.CreateProduct(t.Context(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got.ID)
			assert.Equal(t, "Widget", got.Name)
			assert.Equal(t, int64(500), got.PriceCents)
			assert.Equal(t, 3, got.Stock)
		})
	}
}

func TestCreateProductCategoryNotFound(t *testing.T) {
	m := &countingMetrics{}
	svc := NewService(memory.New(), fakeS3Client{}, fakeCategoryProvider{err: domain.ErrCategoryNotFound}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err := svc.CreateProduct(t.Context(), CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"})
	require.ErrorIs(t, err, domain.ErrCategoryNotFound)
	assert.Equal(t, []string{metrics.StatusError}, m.created)
}

func TestCreateProductPersistenceFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	m := &countingMetrics{}
	svc := NewService(failingRepo{err: boom}, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err := svc.CreateProduct(t.Context(), CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.created)
}

func TestCreateProductRecordsSuccess(t *testing.T) {
	m := &countingMetrics{}
	svc := NewService(memory.New(), fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err := svc.CreateProduct(t.Context(), CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"})
	require.NoError(t, err)
	assert.Equal(t, []string{metrics.StatusSuccess}, m.created)
}
