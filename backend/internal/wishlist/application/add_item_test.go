package application

import (
	"minimart/internal/wishlist/domain"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddItem(t *testing.T) {
	t.Run("Empty user ID is rejected", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})

	t.Run("Empty product ID is rejected", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrProductIDRequired)
	})

	t.Run("Product does not exist", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "missing-prod"})
		require.Error(t, err)
		assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
	})

	t.Run("Product already in wishlist", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductAlreadyInWishlist)
	})

	t.Run("Product added to existing wishlist", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		wishlist, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-2"})
		require.NoError(t, err)
		require.Len(t, wishlist.Items, 2)
		assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)
		assert.Equal(t, "prod-2", wishlist.Items[1].ProductID)
	})

	t.Run("Wishlist created on first add", func(t *testing.T) {
		svc, _, _, _ := newTestService(t)

		wishlist, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.NoError(t, err)
		require.Len(t, wishlist.Items, 1)
		assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)
	})
}
