package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetOrCreateCart(t *testing.T) {
	_, _, _, _, _, svc := newTestService()

	t.Run("creates new cart", func(t *testing.T) {
		cart, err := svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: "user-1"})
		require.NoError(t, err)
		assert.Equal(t, "cust-user-1", cart.CustomerID)
		assert.Equal(t, domain.StatusActive, cart.Status)
	})

	t.Run("returns existing cart", func(t *testing.T) {
		cart1, _ := svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: "user-1"})
		cart2, _ := svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: "user-1"})
		assert.Equal(t, cart1.ID, cart2.ID)
	})

	t.Run("empty userID returns error", func(t *testing.T) {
		_, err := svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})
}
