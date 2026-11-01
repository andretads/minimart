//go:build integration

package test

import (
	"minimart/internal/wishlist/application"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearWishlistIntegration(t *testing.T) {
	svc := newWishlistService(t)

	t.Run("Empty user ID is rejected", func(t *testing.T) {
		err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIDRequired)
	})

	t.Run("All items removed", func(t *testing.T) {
		user := "user-clear-all"
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-2"})
		err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: user})
		require.NoError(t, err)

		wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Clearing an already empty wishlist succeeds", func(t *testing.T) {
		user := "user-clear-empty"
		_, _ = svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
		err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: user})
		require.NoError(t, err)

		wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: user})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Clearing a missing wishlist succeeds", func(t *testing.T) {
		user := "user-clear-missing"
		err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: user})
		require.NoError(t, err)
	})
}
