package application

import (
	"minimart/internal/wishlist/domain"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_AddItem(t *testing.T) {
	t.Run("adds new item to wishlist", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		wishlist, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.NoError(t, err)
		require.Len(t, wishlist.Items, 1)
		assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)
	})

	t.Run("returns error for duplicate product", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductAlreadyInWishlist)
	})

	t.Run("product not found returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-999"})
		require.Error(t, err)
		assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
	})

	t.Run("blank user_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "", ProductID: "prod-1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIDRequired)
	})

	t.Run("blank product_id returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrProductIDRequired)
	})
}
