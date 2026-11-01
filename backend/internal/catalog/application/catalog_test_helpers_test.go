package application

import (
	"context"
	"io"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"
)

// catalogRepoStub satisfies the rest of domain.Repository for use in tests.
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
func (catalogRepoStub) ReleaseStock(context.Context, string) error                  { return nil }
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

type fakeS3Client struct{}

func (fakeS3Client) Upload(context.Context, string, io.Reader, string, int64) error { return nil }
func (fakeS3Client) Delete(context.Context, string) error                           { return nil }
func (fakeS3Client) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "https://presigned.example.com/", nil
}
func (fakeS3Client) Bucket() string { return "test-bucket" }

// newTestService builds a Service backed by the in-memory repository for unit tests.
func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()), repo
}

func validCreateProductRequest() CreateProductRequest {
	return CreateProductRequest{Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1"}
}

func validUpdateProductRequest(id string, version int) UpdateProductRequest {
	return UpdateProductRequest{ID: id, Name: "Gadget", PriceCents: 750, Version: version, CategoryID: "cat-1"}
}

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
