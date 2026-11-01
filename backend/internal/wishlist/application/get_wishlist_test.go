package application

import (
	"minimart/internal/wishlist/domain"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWishlist(t *testing.T) {
	t.Run("Empty user ID is rejected", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})

	t.Run("Existing wishlist returned", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-2"})

		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", wishlist.CustomerID)
		require.Len(t, wishlist.Items, 2)
		assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)
		assert.Equal(t, "prod-2", wishlist.Items[1].ProductID)
	})

	t.Run("Missing wishlist is created empty", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", wishlist.CustomerID)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Concurrent create returns the existing wishlist", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		type result struct {
			wishlist domain.Wishlist
			err      error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		for range 2 {
			go func() {
				defer wg.Done()
				w, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
				results <- result{wishlist: w, err: err}
			}()
		}
		wg.Wait()
		close(results)

		var ids []string
		for r := range results {
			require.NoError(t, r.err)
			ids = append(ids, r.wishlist.ID)
		}
		require.Len(t, ids, 2)
		assert.Equal(t, ids[0], ids[1])
	})
}
