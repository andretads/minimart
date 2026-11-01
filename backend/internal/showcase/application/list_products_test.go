package application

import (
	"minimart/internal/showcase/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	t.Run("Empty catalog returns an empty paginated list", func(t *testing.T) {
		svc, _ := newTestService(t)

		result, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 20})
		require.NoError(t, err)
		assert.Empty(t, result.Items)
		assert.False(t, result.HasMore)
		assert.Empty(t, result.NextCursor)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("Seeded catalog returns products newest-first with cursor metadata", func(t *testing.T) {
		svc, repo := newTestService(t)
		now := time.Now().UTC()
		for i, name := range []string{"Product A", "Product B", "Product C"} {
			seedProduct(t, repo, domain.Product{
				ID:         "p" + string(rune('1'+i)),
				Name:       name,
				PriceCents: int64((i + 1) * 1000),
				CreatedAt:  now.Add(time.Duration(i) * time.Second),
			})
		}

		result, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 20})
		require.NoError(t, err)
		require.Len(t, result.Items, 3)
		assert.Equal(t, "Product C", result.Items[0].Name)
		assert.Equal(t, 20, result.PageSize)
		assert.False(t, result.HasMore)
	})

	t.Run("Products with images expose the primary image URL", func(t *testing.T) {
		svc, repo := newTestService(t)
		now := time.Now().UTC()
		seedProduct(t, repo, domain.Product{ID: "p1", Name: "Product A", PriceCents: 1000, CreatedAt: now})
		seedImage(t, repo, domain.ProductImage{ID: "i1", ProductID: "p1", ObjectKey: "key-1", Position: 1})
		seedImage(t, repo, domain.ProductImage{ID: "i2", ProductID: "p1", ObjectKey: "key-2", Position: 2})

		result, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 20})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Len(t, result.Items[0].Images, 2)
		assert.Equal(t, "key-1", result.Items[0].Images[0].ObjectKey)
	})

	t.Run("Page_size limits number of items returned", func(t *testing.T) {
		svc, repo := newTestService(t)
		now := time.Now().UTC()
		for i := range 3 {
			seedProduct(t, repo, domain.Product{
				ID:         "p" + string(rune('1'+i)),
				Name:       "Product " + string(rune('A'+i)),
				PriceCents: int64((i + 1) * 1000),
				CreatedAt:  now.Add(time.Duration(i) * time.Second),
			})
		}

		result, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 2})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 2, result.PageSize)
		assert.True(t, result.HasMore)
		assert.NotEmpty(t, result.NextCursor)
	})

	t.Run("Cursor advances to the next page", func(t *testing.T) {
		svc, repo := newTestService(t)
		now := time.Now().UTC()
		for i := range 3 {
			seedProduct(t, repo, domain.Product{
				ID:         "p" + string(rune('1'+i)),
				Name:       "Product " + string(rune('A'+i)),
				PriceCents: int64((i + 1) * 1000),
				CreatedAt:  now.Add(time.Duration(i) * time.Second),
			})
		}

		first, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 2})
		require.NoError(t, err)
		require.True(t, first.HasMore)

		second, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 2, Cursor: first.NextCursor})
		require.NoError(t, err)
		assert.Len(t, second.Items, 1)
		assert.False(t, second.HasMore)
		assert.Empty(t, second.NextCursor)
	})

	t.Run("Invalid page_size returns 400", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 101})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)

		_, err = svc.ListProducts(t.Context(), ListProductsRequest{PageSize: -1})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)
	})

	t.Run("Invalid cursor returns 400", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListProducts(t.Context(), ListProductsRequest{PageSize: 2, Cursor: "not-a-cursor"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCursor)
	})
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
