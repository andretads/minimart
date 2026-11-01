package module

import (
	"context"
	"errors"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/presentation/module/client"
	"minimart/internal/catalog/testkit"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// releaseErrRepo makes ReleaseStock fail so the pass-through branch can be
// exercised.
type releaseErrRepo struct {
	catalogRepoStub
	err error
}

type catalogRepoStub struct{}

func (catalogRepoStub) Save(context.Context, domain.Product) error { return nil }
func (catalogRepoStub) Update(context.Context, domain.Product) (domain.Product, error) {
	return domain.Product{}, nil
}

func (catalogRepoStub) GetByID(context.Context, string) (domain.Product, error) {
	return domain.Product{}, domain.ErrProductNotFound
}
func (catalogRepoStub) List(context.Context) ([]domain.Product, error) { return nil, nil }
func (catalogRepoStub) ListByCategory(context.Context, string) ([]domain.Product, error) {
	return nil, nil
}

func (catalogRepoStub) ReserveStock(context.Context, string, []domain.ReservationItem) error {
	return nil
}

func (catalogRepoStub) SaveReview(context.Context, domain.Review) error             { return nil }
func (catalogRepoStub) GetReviews(context.Context, string) ([]domain.Review, error) { return nil, nil }

func (catalogRepoStub) GetReviewsPaginated(context.Context, string, int, int) ([]domain.Review, error) {
	return nil, nil
}
func (catalogRepoStub) CountReviews(context.Context, string) (int, error) { return 0, nil }

func (catalogRepoStub) GetReviewByID(context.Context, string) (domain.Review, error) {
	return domain.Review{}, domain.ErrProductReviewNotFound
}
func (catalogRepoStub) UpdateReviewStats(context.Context, string, int64, int64) error { return nil }
func (catalogRepoStub) GetRatingSummary(context.Context, string) (domain.RatingSummary, error) {
	return domain.RatingSummary{}, nil
}

func (catalogRepoStub) UpdateRatingSummary(context.Context, string, int64, int64) error { return nil }

func (catalogRepoStub) UpdateReviewModeration(context.Context, string, moderation.Result) error {
	return nil
}

func (catalogRepoStub) SaveProductImage(context.Context, domain.ProductImage) (domain.ProductImage, error) {
	return domain.ProductImage{}, nil
}

func (catalogRepoStub) DeleteProductImage(context.Context, string, string) (domain.ProductImage, error) {
	return domain.ProductImage{}, nil
}

func (catalogRepoStub) ListProductImages(context.Context, string) ([]domain.ProductImage, error) {
	return nil, nil
}

func (catalogRepoStub) ListProductImagesByProductIDs(context.Context, []string) (map[string][]domain.ProductImage, error) {
	return nil, nil
}

func (f releaseErrRepo) ReleaseStock(context.Context, string) error { return f.err }

func TestReleaseStock(t *testing.T) {
	t.Run("release restores the held stock", func(t *testing.T) {
		repo := memory.New()
		widget, err := domain.NewProduct("p1", "Widget", 500, 1, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), widget))
		adapter := New(application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		require.NoError(t, adapter.ReserveStock(t.Context(), "o1", []client.LineItem{{ProductID: "p1", Quantity: 1}}))
		require.NoError(t, adapter.ReleaseStock(t.Context(), "o1"))

		got, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Stock)
	})

	t.Run("releasing twice is idempotent", func(t *testing.T) {
		repo := memory.New()
		widget, err := domain.NewProduct("p1", "Widget", 500, 1, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), widget))
		adapter := New(application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		require.NoError(t, adapter.ReserveStock(t.Context(), "o1", []client.LineItem{{ProductID: "p1", Quantity: 1}}))
		require.NoError(t, adapter.ReleaseStock(t.Context(), "o1"))
		require.NoError(t, adapter.ReleaseStock(t.Context(), "o1"))

		got, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Stock, "a redelivered release does not add stock twice")
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		adapter := New(application.NewService(releaseErrRepo{err: boom}, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

		assert.ErrorIs(t, adapter.ReleaseStock(t.Context(), "o1"), boom)
	})
}
