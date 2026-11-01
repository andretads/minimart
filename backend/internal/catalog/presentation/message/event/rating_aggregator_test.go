//go:build unit

package event

import (
	"encoding/json"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/internal/catalog/testkit"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_AggregateRating_Success(t *testing.T) {
	repo := memory.New()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
	require.NoError(t, repo.SaveReview(t.Context(), domain.Review{ID: "r1", ProductID: "p1", CustomerID: "c1", Description: "Good", Rating: 5, Status: "approved", CreatedAt: time.Now().UTC()}))

	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewModerated{ReviewID: "r1", ProductID: "p1", Decision: "approved", Rating: 5})
	require.NoError(t, err)

	err = h.AggregateRating(t.Context(), msg)
	require.NoError(t, err)

	prod, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), prod.TotalRating)
	assert.Equal(t, int64(1), prod.TotalReviews)
}

func TestHandler_AggregateRating_InvalidMessage(t *testing.T) {
	repo := memory.New()
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	h := NewHandler(svc)

	err := h.AggregateRating(t.Context(), []byte("{bad"))
	assert.Error(t, err)
}

func TestHandler_AggregateRating_ProductNotFound(t *testing.T) {
	repo := memory.New()
	svc := application.NewService(repo, testkit.NewFakeS3Client(), fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	h := NewHandler(svc)

	msg, err := json.Marshal(message.ReviewModerated{ReviewID: "r1", ProductID: "missing", Decision: "approved", Rating: 5})
	require.NoError(t, err)

	err = h.AggregateRating(t.Context(), msg)
	assert.Error(t, err)
}
