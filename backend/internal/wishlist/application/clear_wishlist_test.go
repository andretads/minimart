package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ClearWishlist(t *testing.T) {
	t.Run("clears all items from wishlist", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-2"})
		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)

		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("clears empty wishlist is no-op", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
	})

	t.Run("blank user_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		err := svc.ClearWishlist(t.Context(), ClearWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})
}
