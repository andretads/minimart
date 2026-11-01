package application

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestService_ClearCart(t *testing.T) {
	t.Run("clears cart", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, _ = svc.AddItem(t.Context(), AddItemRequest{UserID: "user-1", ProductID: "prod-1", Quantity: 2})
		_, err := svc.ClearCart(t.Context(), ClearCartRequest{UserID: "user-1"})
		require.NoError(t, err)
	})
}
