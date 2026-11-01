package application

import (
	"context"
	"io"
	"minimart/internal/catalog/domain"
	"minimart/pkg/moderation"
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
