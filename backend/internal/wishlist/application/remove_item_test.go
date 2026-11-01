package application

import (
	"minimart/internal/wishlist/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_RemoveItem(t *testing.T) {
	t.Run("removes existing item from wishlist", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		wishlist, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.NoError(t, err)
		assert.Empty(t, wishlist.Items)
	})

	t.Run("returns error for non-existent product", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "user-1", ProductID: "prod-999"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotInWishlist)
	})

	t.Run("blank user_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})

	t.Run("blank product_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "user-1", ProductID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrProductIDRequired)
	})
}
