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

func seedReviewProduct(t *testing.T, repo *memory.Repository) {
	t.Helper()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

func TestAddReview(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "missing", CustomerID: "cust-1", Description: "Good", Rating: 4})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Invalid rating too low is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 0})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidRating)
	})

	t.Run("Invalid rating too high is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Bad", Rating: 6})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidRating)
	})

	t.Run("Empty description is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "   ", Rating: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Missing customer is rejected", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "", Description: "Good", Rating: 4})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		t.Skip("handled at HTTP layer")
	})

	t.Run("Storage failure", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &reviewErrorRepoWithProduct{
			products: map[string]domain.Product{"p1": {ID: "p1", Name: "Widget", PriceCents: 500, Stock: 3, CategoryID: "cat-1", Version: 1}},
			err:      boom,
		}
		m := &countingMetrics{}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, m, moderation.NewNoopProvider())

		_, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Good", Rating: 4})
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("Review got added with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)

		assert.NotEmpty(t, review.ID)
		assert.Equal(t, "p1", review.ProductID)
		assert.Equal(t, "cust-1", review.CustomerID)
		assert.Equal(t, "Great product!", review.Description)
		assert.Equal(t, 5, review.Rating)
		assert.False(t, review.CreatedAt.IsZero())
	})

	t.Run("Review starts as pending for moderation", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusPending, review.Status)
	})

	t.Run("Review is attributed to the authenticated customer", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedReviewProduct(t, repo)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)
		assert.Equal(t, "cust-1", review.CustomerID)
	})

	t.Run("Review is approved by moderation", func(t *testing.T) {
		repo := memory.New()
		seedReviewProduct(t, repo)
		provider := &testModerationProvider{result: moderation.Result{
			Decision: "approved", Confidence: 0.95, Categories: []string{}, Explanation: "ok",
			Model: "m", ModelVersion: "1", ModeratedAt: time.Now().UTC(),
		}}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Great product!", Rating: 5})
		require.NoError(t, err)

		err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusApproved, updated.Status)
		assert.NotEmpty(t, updated.ModerationDecision)
	})

	t.Run("Review is rejected by moderation", func(t *testing.T) {
		repo := memory.New()
		seedReviewProduct(t, repo)
		provider := &testModerationProvider{result: moderation.Result{
			Decision: "rejected", Confidence: 0.9, Categories: []string{"profanity"}, Explanation: "bad",
			Model: "m", ModelVersion: "1", ModeratedAt: time.Now().UTC(),
		}}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "This product is f***ing garbage!", Rating: 1})
		require.NoError(t, err)

		err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusRejected, updated.Status)

		reviews, err := svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		assert.NotContains(t, reviews, review.ID)
	})

	t.Run("Review needs human review", func(t *testing.T) {
		repo := memory.New()
		seedReviewProduct(t, repo)
		provider := &testModerationProvider{result: moderation.Result{
			Decision: "needs_review", Confidence: 0.5, Categories: []string{}, Explanation: "unsure",
			Model: "m", ModelVersion: "1", ModeratedAt: time.Now().UTC(),
		}}
		svc := NewService(repo, fakeS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, provider)

		review, err := svc.AddReview(t.Context(), AddReviewRequest{ProductID: "p1", CustomerID: "cust-1", Description: "Hmm, not sure about this one", Rating: 3})
		require.NoError(t, err)

		err = svc.ModerateReview(t.Context(), ModerateReviewRequest{ReviewID: review.ID})
		require.NoError(t, err)

		updated, err := repo.GetReviewByID(t.Context(), review.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReviewStatusNeedsReview, updated.Status)

		reviews, err := svc.GetReviews(t.Context(), "p1")
		require.NoError(t, err)
		assert.NotContains(t, reviews, review.ID)
	})
}
