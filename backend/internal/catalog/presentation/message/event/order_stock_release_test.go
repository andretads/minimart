package event

import (
	"context"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/testkit"
	"minimart/internal/category/presentation/module/client"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCategoryProvider is a test double for CategoryProvider.
type fakeCategoryProvider struct {
	err error
}

func (f fakeCategoryProvider) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	if f.err != nil {
		return client.CategoryDTO{}, f.err
	}
	return client.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

// seededHandler returns a handler backed by a memory repository holding one
// product (p1) whose stock is already reserved by order o1.
func seededHandler(t *testing.T) (*Handler, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 5, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
	require.NoError(t, repo.ReserveStock(t.Context(), "o1", []domain.ReservationItem{{ProductID: "p1", Quantity: 2}}))
	return NewHandler(application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())), repo
}

func TestOrderStockRelease(t *testing.T) {
	h, repo := seededHandler(t)

	require.NoError(t, h.OrderStockRelease(t.Context(), []byte(`{"order_id":"o1"}`)))

	p, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, 5, p.Stock, "the reservation is returned to the product")
}

func TestOrderStockReleaseIsIdempotent(t *testing.T) {
	h, repo := seededHandler(t)
	msg := []byte(`{"order_id":"o1"}`)

	require.NoError(t, h.OrderStockRelease(t.Context(), msg))
	require.NoError(t, h.OrderStockRelease(t.Context(), msg))

	p, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, 5, p.Stock, "a redelivered release does not add stock twice")
}

func TestOrderStockReleaseMalformedJSON(t *testing.T) {
	h := NewHandler(nil)
	assert.Error(t, h.OrderStockRelease(t.Context(), []byte("not json")))
}
