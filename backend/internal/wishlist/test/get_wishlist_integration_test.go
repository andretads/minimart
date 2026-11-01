//go:build integration

package test

import (
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWishlistIntegration(t *testing.T) {
	svc := newWishlistService(t)

	t.Run("Empty user ID is rejected", func(t *testing.T) {
		_, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIDRequired)
	})

	t.Run("Existing wishlist returned", func(t *testing.T) {
		user := "user-existing-wl"
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-2"})

		wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
		require.NoError(t, err)
		assert.Equal(t, "cust-"+user, wishlist.CustomerID)
		require.Len(t, wishlist.Items, 2)
	})

	t.Run("Missing wishlist is created empty", func(t *testing.T) {
		user := "user-missing-wl"
		wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
		require.NoError(t, err)
		assert.Equal(t, "cust-"+user, wishlist.CustomerID)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Concurrent create returns the existing wishlist", func(t *testing.T) {
		user := "user-concurrent-wl"
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
				w, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
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
