package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_UpdateItemQuantity(t *testing.T) {
	t.Run("updates quantity", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		cart, err := svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 5})
		require.NoError(t, err)
		assert.Equal(t, 5, cart.Items[0].Quantity)
	})

	t.Run("item not found", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		_, err := svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{UserID: "user-1", ProductID: "prod-999", Quantity: 5})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})
}
