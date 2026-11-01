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

// reviewErrorRepoWithProduct has a product but fails on SaveReview
type reviewErrorRepoWithProduct struct {
	catalogRepoStub
	err      error
	products map[string]domain.Product
}

func (f *reviewErrorRepoWithProduct) GetByID(ctx context.Context, id string) (domain.Product, error) {
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return domain.Product{}, domain.ErrProductNotFound
}

func (f *reviewErrorRepoWithProduct) SaveReview(ctx context.Context, rev domain.Review) error {
	return f.err
}

func TestAddReview(t *testing.T) {
	tests := []struct {
		name    string
		req     AddReviewRequest
		wantErr error
	}{
		{"valid", AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5}, nil},
		{"invalid rating too low", AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 0}, domain.ErrInvalidRating},
		{"invalid rating too high", AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 6}, domain.ErrInvalidRating},
		{"empty description", AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "   ", Rating: 5}, domain.ErrEmptyName},
		{"missing customer", AddReviewRequest{ProductID: "p1", CustomerID: "", Description: "Good", Rating: 4}, domain.ErrEmptyCustomerID},
		{"missing product", AddReviewRequest{ProductID: "missing", CustomerID: "cust-1", Description: "Good", Rating: 4}, domain.ErrProductNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := memory.New()
			if tt.name != "missing product" {
				p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
				require.NoError(t, err)
				require.NoError(t, repo.Save(t.Context(), p))
			}

			m := &countingMetrics{}
			svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

			got, err := svc.AddReview(t.Context(), tt.req)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got.ID)
			assert.Equal(t, tt.req.ProductID, got.ProductID)
			assert.Equal(t, tt.req.CustomerID, got.CustomerID)
			assert.Equal(t, tt.req.Description, got.Description)
			assert.Equal(t, tt.req.Rating, got.Rating)
			assert.Equal(t, "pending", got.Status, "new reviews start as pending")
			assert.False(t, got.CreatedAt.IsZero())
		})
	}
}

func TestAddReviewProductNotFound(t *testing.T) {
	repo := memory.New()
	m := &countingMetrics{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "missing", CustomerID: "cust-1", Description: "Good", Rating: 4})
	require.ErrorIs(t, err, domain.ErrProductNotFound)
	assert.Equal(t, []string{metrics.StatusError}, m.updated)
}

func TestAddReviewPersistenceFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	repo := &reviewErrorRepoWithProduct{
		products: map[string]domain.Product{
			"p1": {ID: "p1", Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1", Version: 1},
		},
		err: boom,
	}
	m := &countingMetrics{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Good", Rating: 4})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.updated)
}

func TestAddReviewRecordsSuccess(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	m := &countingMetrics{}
	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

	_, err = svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great!", Rating: 5})
	require.NoError(t, err)
	assert.Equal(t, []string{metrics.StatusSuccess}, m.updated)
}

func TestGetReviews(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	// Add some reviews
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r2", ProductID: "p1", CustomerID: "c2", Description: "Bad", Rating: 2, Status: "approved", CreatedAt: time.Now().UTC()}))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r3", ProductID: "p1", CustomerID: "c3", Description: "Pending", Rating: 3, Status: "pending", CreatedAt: time.Now().UTC()}))

	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	reviews, err := svc.GetReviews(t.Context(), "p1")
	require.NoError(t, err)
	assert.Len(t, reviews, 2, "only approved reviews returned")
	assert.Equal(t, "r1", reviews[0].ID) // first added (by created_at desc)
	assert.Equal(t, "r2", reviews[1].ID)
}

func TestGetReviewSummary(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	p.TotalRating = 20
	p.TotalReviews = 5
	require.NoError(t, repo.Save(t.Context(), p))

	svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	totalRating, totalReviews, avg, err := svc.GetReviewSummary(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(20), totalRating)
	assert.Equal(t, int64(5), totalReviews)
	assert.InDelta(t, 4.0, avg, 0.0001)
}
