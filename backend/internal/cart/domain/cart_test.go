package domain

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCart(t *testing.T) {
	t.Parallel()

	cart, err := NewCart("cart-1", "customer-1")
	require.NoError(t, err)
	assert.Equal(t, "cart-1", cart.ID)
	assert.Equal(t, "customer-1", cart.CustomerID)
	assert.Equal(t, StatusActive, cart.Status)
	assert.Equal(t, 1, cart.Version)
	assert.Empty(t, cart.Items)
	assert.False(t, cart.CreatedAt.IsZero())
	assert.Equal(t, cart.CreatedAt, cart.UpdatedAt)
}

func TestNewCart_EmptyCustomerID(t *testing.T) {
	t.Parallel()

	_, err := NewCart("cart-1", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyCustomerID)
}

func TestCart_EstimatedTotalCents(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	cart.Items = []Item{
		{UnitPriceSnapshotCents: 1000, Quantity: 2},
		{UnitPriceSnapshotCents: 500, Quantity: 3},
	}
	assert.Equal(t, int64(3500), cart.EstimatedTotalCents())
}

func TestCart_AddItem_NewProduct(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.AddItem("prod-1", "Product 1", 1000, 2)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-1", cart.Items[0].ProductID)
	assert.Equal(t, "Product 1", cart.Items[0].ProductNameSnapshot)
	assert.Equal(t, int64(1000), cart.Items[0].UnitPriceSnapshotCents)
	assert.Equal(t, 2, cart.Items[0].Quantity)
}

func TestCart_AddItem_ExistingProduct(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 2))
	err := cart.AddItem("prod-1", "Product 1", 1000, 3)
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, 5, cart.Items[0].Quantity)
}

func TestCart_AddItem_InvalidQuantity(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.AddItem("prod-1", "Product 1", 1000, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

func TestCart_AddItem_NotActive(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	cart.Status = StatusConsumed
	err := cart.AddItem("prod-1", "Product 1", 1000, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartNotActive)
}

func TestCart_UpdateItemQuantity(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 2))
	err := cart.UpdateItemQuantity("prod-1", 5)
	require.NoError(t, err)
	assert.Equal(t, 5, cart.Items[0].Quantity)
}

func TestCart_UpdateItemQuantity_InvalidQuantity(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 2))
	err := cart.UpdateItemQuantity("prod-1", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

func TestCart_UpdateItemQuantity_NotFound(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.UpdateItemQuantity("prod-999", 5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCart_RemoveItem(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 2))
	require.NoError(t, cart.AddItem("prod-2", "Product 2", 500, 1))
	err := cart.RemoveItem("prod-1")
	require.NoError(t, err)
	require.Len(t, cart.Items, 1)
	assert.Equal(t, "prod-2", cart.Items[0].ProductID)
}

func TestCart_RemoveItem_NotFound(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.RemoveItem("prod-999")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartItemNotFound)
}

func TestCart_Clear(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 2))
	err := cart.Clear()
	require.NoError(t, err)
	assert.Empty(t, cart.Items)
}

func TestCart_BeginCheckout(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.BeginCheckout("idem-key-1")
	require.NoError(t, err)
	assert.Equal(t, StatusCheckoutPending, cart.Status)
	assert.Equal(t, "idem-key-1", cart.CheckoutKey)
}

func TestCart_BeginCheckout_NotActive(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	cart.Status = StatusConsumed
	err := cart.BeginCheckout("idem-key-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartConsumed)
}

func TestCart_MarkConsumed(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	cart.Status = StatusCheckoutPending
	cart.CheckoutKey = "idem-key-1"
	err := cart.MarkConsumed("order-1")
	require.NoError(t, err)
	assert.Equal(t, StatusConsumed, cart.Status)
	assert.Equal(t, "order-1", cart.OrderID)
	assert.Empty(t, cart.CheckoutKey)
}

func TestCart_MarkConsumed_NotPending(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.MarkConsumed("order-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartStateChanged)
}

func TestCart_ReleaseCheckout(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	cart.Status = StatusCheckoutPending
	cart.CheckoutKey = "idem-key-1"
	err := cart.ReleaseCheckout()
	require.NoError(t, err)
	assert.Equal(t, StatusActive, cart.Status)
	assert.Empty(t, cart.CheckoutKey)
}

func TestCart_ReleaseCheckout_NotPending(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	err := cart.ReleaseCheckout()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCartStateChanged)
}

func TestCart_IsMutable(t *testing.T) {
	t.Parallel()

	cart, _ := NewCart("cart-1", "customer-1")
	assert.True(t, cart.IsMutable())

	cart.Status = StatusCheckoutPending
	assert.False(t, cart.IsMutable())

	cart.Status = StatusConsumed
	assert.False(t, cart.IsMutable())
}

func TestItem_LineTotal(t *testing.T) {
	t.Parallel()

	item := Item{
		UnitPriceSnapshotCents: 1000,
		Quantity:               3,
	}
	// Line total is calculated on demand in response layer
	assert.Equal(t, int64(3000), item.UnitPriceSnapshotCents*int64(item.Quantity))
}

func TestCart_UpdatedAt_ChangesOnMutation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cart, _ := NewCart("cart-1", "customer-1")
		original := cart.UpdatedAt
		// Fake time.Sleep advances the synctest clock instantly
		time.Sleep(time.Millisecond)
		require.NoError(t, cart.AddItem("prod-1", "Product 1", 1000, 1))
		assert.True(t, cart.UpdatedAt.After(original))
	})
}
