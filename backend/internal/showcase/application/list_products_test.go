package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	products  []domain.Product
	listError error
	reviews   map[string][]domain.Review
}

func (f *fakeRepo) List(ctx context.Context, limit int, after *domain.Cursor) ([]domain.Product, error) {
	if f.listError != nil {
		return nil, f.listError
	}
	// Mirror keyset ordering: created_at DESC, id DESC.
	items := append([]domain.Product(nil), f.products...)
	// Sort newest-first by created_at then id.
	// (tests seed products already in order; keep stable fallback)
	_ = items
	var out []domain.Product
	for _, p := range f.products {
		if after != nil {
			if p.CreatedAt.After(after.CreatedAt) ||
				(p.CreatedAt.Equal(after.CreatedAt) && p.ID >= after.ID) {
				continue
			}
		}
		out = append(out, p)
	}
	// newest-first
	// assume input already newest-first; if not, sort
	_ = time.Now
	if len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []domain.Product{}
	}
	return out, nil
}

func (f *fakeRepo) ListImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	return nil, nil
}

func (f *fakeRepo) ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	return map[string][]domain.ProductImage{}, nil
}

func (f *fakeRepo) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	return f.reviews[productID], nil
}

func (f *fakeRepo) CountReviews(ctx context.Context, productID string) (int, error) {
	return len(f.reviews[productID]), nil
}

func (f *fakeRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id string) (domain.Product, error) {
	for _, p := range f.products {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Product{}, domain.ErrProductNotFound
}

func (f *fakeRepo) ListByCategory(ctx context.Context, categoryID string, limit int, after *domain.Cursor) ([]domain.Product, error) {
	if f.listError != nil {
		return nil, f.listError
	}
	var out []domain.Product
	for _, p := range f.products {
		if p.CategoryID != categoryID {
			continue
		}
		if after != nil {
			if p.CreatedAt.After(after.CreatedAt) ||
				(p.CreatedAt.Equal(after.CreatedAt) && p.ID >= after.ID) {
				continue
			}
		}
		out = append(out, p)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []domain.Product{}
	}
	return out, nil
}

func TestListProducts(t *testing.T) {
	now := time.Now()
	products := []domain.Product{
		{ID: "p1", Name: "Product 1", PriceCents: 100, CreatedAt: now.Add(-4 * time.Hour)},
		{ID: "p2", Name: "Product 2", PriceCents: 200, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: "p3", Name: "Product 3", PriceCents: 300, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "p4", Name: "Product 4", PriceCents: 400, CreatedAt: now.Add(-1 * time.Hour)},
		{ID: "p5", Name: "Product 5", PriceCents: 500, CreatedAt: now},
	}
	// newest-first order as stored
	newestFirst := []domain.Product{products[4], products[3], products[2], products[1], products[0]}

	tests := []struct {
		name        string
		req         ListProductsRequest
		setup       func() *fakeRepo
		wantItems   int
		wantHasMore bool
		wantCursor  bool
		wantErrMsg  string
	}{
		{
			name: "first page default size",
			req:  ListProductsRequest{PageSize: 20},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantItems:   5,
			wantHasMore: false,
			wantCursor:  false,
		},
		{
			name: "page size 2 has more",
			req:  ListProductsRequest{PageSize: 2},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantItems:   2,
			wantHasMore: true,
			wantCursor:  true,
		},
		{
			name: "cursor advances",
			req:  ListProductsRequest{PageSize: 2, Cursor: EncodeCursor(domain.Cursor{CreatedAt: newestFirst[1].CreatedAt, ID: newestFirst[1].ID})},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantItems:   2,
			wantHasMore: true,
			wantCursor:  true,
		},
		{
			name: "last page partial",
			req:  ListProductsRequest{PageSize: 2, Cursor: EncodeCursor(domain.Cursor{CreatedAt: newestFirst[3].CreatedAt, ID: newestFirst[3].ID})},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantItems:   1,
			wantHasMore: false,
			wantCursor:  false,
		},
		{
			name: "empty catalog",
			req:  ListProductsRequest{PageSize: 20},
			setup: func() *fakeRepo {
				return &fakeRepo{products: []domain.Product{}}
			},
			wantItems:   0,
			wantHasMore: false,
			wantCursor:  false,
		},
		{
			name: "invalid page_size zero",
			req:  ListProductsRequest{PageSize: 0},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			// PageSize 0 defaults to 20 -> valid
			wantItems:   5,
			wantHasMore: false,
		},
		{
			name: "invalid page_size over max",
			req:  ListProductsRequest{PageSize: 101},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantErrMsg: "tamanho da página deve estar entre 1 e 100",
		},
		{
			name: "invalid page_size negative",
			req:  ListProductsRequest{PageSize: -1},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantErrMsg: "tamanho da página deve estar entre 1 e 100",
		},
		{
			name: "invalid cursor",
			req:  ListProductsRequest{PageSize: 2, Cursor: "not-a-cursor"},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst}
			},
			wantErrMsg: "cursor inválido",
		},
		{
			name: "repo list error propagates",
			req:  ListProductsRequest{PageSize: 20},
			setup: func() *fakeRepo {
				return &fakeRepo{products: newestFirst, listError: errors.New("db error")}
			},
			wantErrMsg: "db error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.setup(), nil, 0)
			result, err := svc.ListProducts(t.Context(), tc.req)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.Items, tc.wantItems)
			require.Equal(t, tc.wantHasMore, result.HasMore)
			if tc.wantCursor {
				require.NotEmpty(t, result.NextCursor)
			} else {
				require.Empty(t, result.NextCursor)
			}
		})
	}
}

func TestDecodeCursorRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	c := domain.Cursor{CreatedAt: now, ID: "abc"}
	tok := EncodeCursor(c)
	got, err := DecodeCursor(tok)
	require.NoError(t, err)
	require.Equal(t, c.ID, got.ID)
	require.True(t, c.CreatedAt.Equal(got.CreatedAt))
}
