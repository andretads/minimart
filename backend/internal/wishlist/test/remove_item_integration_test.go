//go:build integration

package test

import (
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveItemIntegration(t *testing.T) {
	svc := newWishlistService(t)

	t.Run("Empty user ID is rejected", func(t *testing.T) {
		_, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIDRequired)
	})

	t.Run("Empty product ID is rejected", func(t *testing.T) {
		_, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "user-1", ProductID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrProductIDRequired)
	})

	t.Run("Wishlist not found", func(t *testing.T) {
		_, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "user-notfound", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrWishlistNotFound)
	})

	t.Run("Product not in wishlist", func(t *testing.T) {
		user := "user-notin"
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-1"})
		_, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: user, ProductID: "prod-2"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotInWishlist)
	})

	t.Run("Product removed", func(t *testing.T) {
		user := "user-remove"
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-2"})
		wishlist, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: user, ProductID: "prod-1"})
		require.NoError(t, err)
		require.Len(t, wishlist.Items, 1)
		assert.Equal(t, "prod-2", wishlist.Items[0].ProductID)
	})

	t.Run("Other items remain", func(t *testing.T) {
		user := "user-other"
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-1"})
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-2"})
		_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: user, ProductID: "prod-3"})
		wishlist, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: user, ProductID: "prod-2"})
		require.NoError(t, err)
		require.Len(t, wishlist.Items, 2)
	})
}
