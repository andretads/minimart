package module

import (
	"context"
	"errors"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/presentation/module/client"
	"minimart/internal/catalog/testkit"
	categoryclient "minimart/internal/category/presentation/module/client"
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

func (f fakeCategoryProvider) GetCategory(ctx context.Context, id string) (categoryclient.CategoryDTO, error) {
	if f.err != nil {
		return categoryclient.CategoryDTO{}, f.err
	}
	return categoryclient.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

// failingRepo makes GetByID fail with a non-not-found error so the
// pass-through branch can be exercised.
type failingRepo struct {
	err error
}

func (f failingRepo) Save(context.Context, domain.Product) error { return nil }
func (f failingRepo) Update(context.Context, domain.Product) (domain.Product, error) {
	return domain.Product{}, nil
}

func (f failingRepo) GetByID(context.Context, string) (domain.Product, error) {
	return domain.Product{}, f.err
}
func (f failingRepo) List(context.Context) ([]domain.Product, error) { return nil, nil }
func (f failingRepo) ListByCategory(context.Context, string) ([]domain.Product, error) {
	return nil, nil
}

func (f failingRepo) ReserveStock(context.Context, string, []domain.ReservationItem) error {
	return nil
}
func (f failingRepo) ReleaseStock(context.Context, string) error                  { return nil }
func (f failingRepo) SaveReview(context.Context, domain.Review) error             { return nil }
func (f failingRepo) GetReviews(context.Context, string) ([]domain.Review, error) { return nil, nil }

func (f failingRepo) GetReviewsPaginated(context.Context, string, int, int) ([]domain.Review, error) {
	return nil, nil
}
func (f failingRepo) CountReviews(context.Context, string) (int, error) { return 0, nil }

func (f failingRepo) GetReviewByID(context.Context, string) (domain.Review, error) {
	return domain.Review{}, domain.ErrProductReviewNotFound
}
func (f failingRepo) UpdateReviewStats(context.Context, string, int64, int64) error { return nil }
func (f failingRepo) GetRatingSummary(context.Context, string) (domain.RatingSummary, error) {
	return domain.RatingSummary{}, nil
}
func (f failingRepo) UpdateRatingSummary(context.Context, string, int64, int64) error { return nil }
func (f failingRepo) UpdateReviewModeration(context.Context, string, moderation.Result) error {
	return nil
}

func (f failingRepo) SaveProductImage(context.Context, domain.ProductImage) (domain.ProductImage, error) {
	return domain.ProductImage{}, nil
}

func (f failingRepo) DeleteProductImage(context.Context, string, string) (domain.ProductImage, error) {
	return domain.ProductImage{}, nil
}

func (f failingRepo) ListProductImages(context.Context, string) ([]domain.ProductImage, error) {
	return nil, nil
}

func (f failingRepo) ListProductImagesByProductIDs(context.Context, []string) (map[string][]domain.ProductImage, error) {
	return nil, nil
}

func TestGetProduct(t *testing.T) {
	widget, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)

	t.Run("maps a domain product into the client type", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), widget))
		adapter := New(application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		got, err := adapter.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, client.Product{ID: "p1", Name: "Widget", PriceCents: 500}, got)
	})

	t.Run("translates not-found into the client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		_, err := adapter.GetProduct(t.Context(), "missing")
		assert.ErrorIs(t, err, client.ErrProductNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(failingRepo{err: boom}, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		_, err := adapter.GetProduct(t.Context(), "p1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, client.ErrProductNotFound)
	})
}
