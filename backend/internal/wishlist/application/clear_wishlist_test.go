package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearWishlist(t *testing.T) {
	t.Run("Empty user ID is rejected", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})

	t.Run("All items removed", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-2"})
		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)

		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Clearing an already empty wishlist succeeds", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, _ = svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)

		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("Clearing a missing wishlist succeeds", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
	})
}
