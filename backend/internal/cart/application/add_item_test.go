package application

import (
	"minimart/internal/cart/domain"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_AddItem(t *testing.T) {
	t.Run("adds new item", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		cart, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, "prod-1", cart.Items[0].ProductID)
		assert.Equal(t, 2, cart.Items[0].Quantity)
	})

	t.Run("increments existing item", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		cart, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 3})
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, 5, cart.Items[0].Quantity)
	})

	t.Run("produto não encontrado", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-999", Quantity: 1})
		require.Error(t, err)
		assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
	})

	t.Run("invalid quantity", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 0})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})
}
