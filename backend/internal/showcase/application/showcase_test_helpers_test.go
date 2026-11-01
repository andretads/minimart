package application

import (
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/memory"
	"testing"
	"time"
)

func newTestService(t *testing.T) (*Service, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	return NewService(repo, nil, 0), repo
}

func seedProduct(t *testing.T, repo *memory.Repository, p domain.Product) {
	t.Helper()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	requireNoErr(t, repo.Save(p))
}

func seedImage(t *testing.T, repo *memory.Repository, img domain.ProductImage) {
	t.Helper()
	requireNoErr(t, repo.SaveImage(img))
}

func seedReview(t *testing.T, repo *memory.Repository, r domain.Review) {
	t.Helper()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	requireNoErr(t, repo.SaveReview(r))
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
