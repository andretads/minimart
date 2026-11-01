package application

import (
	"minimart/internal/cart/domain"
	"testing"

	customerclient "minimart/internal/customer/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetCart(t *testing.T) {
	t.Run("returns existing cart", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		// First create a cart via GetOrCreateCart
		_, _ = svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: "user-1"})
		cart, err := svc.GetCart(t.Context(), GetCartRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", cart.CustomerID)
		assert.Equal(t, domain.StatusActive, cart.Status)
	})

	t.Run("returns error when customer not found", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.GetCart(t.Context(), GetCartRequest{UserID: "user-nonexistent"})
		require.Error(t, err)
		assert.ErrorIs(t, err, customerclient.ErrCustomerNotFound)
	})

	t.Run("empty userID returns error", func(t *testing.T) {
		_, _, _, _, _, svc := newTestService()
		_, err := svc.GetCart(t.Context(), GetCartRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})
}
