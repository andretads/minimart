package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_RemoveItem(t *testing.T) {
	t.Run("removes item", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		cart, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "user-1", ProductID: "prod-1"})
		require.NoError(t, err)
		assert.Empty(t, cart.Items)
	})

	t.Run("item not found", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		_, err := svc.RemoveItem(t.Context(), RemoveItemRequest{UserID: "user-1", ProductID: "prod-999"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})
}
