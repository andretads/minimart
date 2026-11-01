package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetWishlist(t *testing.T) {
	t.Run("creates new wishlist for user", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", wishlist.CustomerID)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("returns existing wishlist", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		wishlist, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", wishlist.CustomerID)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("blank user_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.GetWishlist(t.Context(), GetWishlistRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})
}
