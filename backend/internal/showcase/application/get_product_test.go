package application

import (
	"context"
	"errors"
	"minimart/internal/showcase/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

type getProductFakeRepo struct {
	products map[string]domain.Product
	err      error
}

func (f *getProductFakeRepo) List(ctx context.Context, limit int, after *domain.Cursor) ([]domain.Product, error) {
	return nil, nil
}

func (f *getProductFakeRepo) ListByCategory(ctx context.Context, categoryID string, limit int, after *domain.Cursor) ([]domain.Product, error) {
	return nil, nil
}

func (f *getProductFakeRepo) ListImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	return nil, nil
}

func (f *getProductFakeRepo) ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	return map[string][]domain.ProductImage{}, nil
}

func (f *getProductFakeRepo) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	return nil, nil
}

func (f *getProductFakeRepo) CountReviews(ctx context.Context, productID string) (int, error) {
	return 0, nil
}

func (f *getProductFakeRepo) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}

func (f *getProductFakeRepo) GetByID(ctx context.Context, id string) (domain.Product, error) {
	if f.err != nil {
		return domain.Product{}, f.err
	}
	if p, ok := f.products[id]; ok {
		return p, nil
	}
	return domain.Product{}, domain.ErrProductNotFound
}

func TestGetProduct(t *testing.T) {
	product := domain.Product{ID: "p1", Name: "Product 1", PriceCents: 100}

	tests := []struct {
		name       string
		setup      func() *getProductFakeRepo
		id         string
		want       domain.Product
		wantErrMsg string
	}{
		{
			name: "found",
			setup: func() *getProductFakeRepo {
				return &getProductFakeRepo{products: map[string]domain.Product{product.ID: product}}
			},
			id:   product.ID,
			want: product,
		},
		{
			name: "not found",
			setup: func() *getProductFakeRepo {
				return &getProductFakeRepo{products: map[string]domain.Product{}}
			},
			id:         "missing",
			wantErrMsg: "produto não encontrado",
		},
		{
			name: "repo error propagates",
			setup: func() *getProductFakeRepo {
				return &getProductFakeRepo{err: errors.New("db error")}
			},
			id:         "p1",
			wantErrMsg: "db error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.setup(), nil, 0)
			got, err := svc.GetProduct(t.Context(), tc.id)
			if tc.wantErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
