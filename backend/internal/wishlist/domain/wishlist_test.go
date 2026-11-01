package domain

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWishlist(t *testing.T) {
	t.Parallel()

	wishlist, err := NewWishlist("wl-1", "customer-1")
	require.NoError(t, err)
	assert.Equal(t, "wl-1", wishlist.ID)
	assert.Equal(t, "customer-1", wishlist.CustomerID)
	assert.Empty(t, wishlist.Items)
	assert.False(t, wishlist.CreatedAt.IsZero())
	assert.Equal(t, wishlist.CreatedAt, wishlist.UpdatedAt)
}

func TestNewWishlist_EmptyCustomerID(t *testing.T) {
	t.Parallel()

	_, err := NewWishlist("wl-1", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyCustomerID)
}

func TestWishlist_AddItem_NewProduct(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	err := wishlist.AddItem("prod-1")
	require.NoError(t, err)
	require.Len(t, wishlist.Items, 1)
	assert.Equal(t, "prod-1", wishlist.Items[0].ProductID)
	assert.False(t, wishlist.Items[0].AddedAt.IsZero())
}

func TestWishlist_AddItem_DuplicateProduct(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	require.NoError(t, wishlist.AddItem("prod-1"))
	err := wishlist.AddItem("prod-1")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProductAlreadyInWishlist)
	require.Len(t, wishlist.Items, 1)
}

func TestWishlist_RemoveItem_ExistingProduct(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	require.NoError(t, wishlist.AddItem("prod-1"))
	require.NoError(t, wishlist.AddItem("prod-2"))
	err := wishlist.RemoveItem("prod-1")
	require.NoError(t, err)
	require.Len(t, wishlist.Items, 1)
	assert.Equal(t, "prod-2", wishlist.Items[0].ProductID)
}

func TestWishlist_RemoveItem_NotFound(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	err := wishlist.RemoveItem("prod-999")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotInWishlist)
}

func TestWishlist_Clear(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	require.NoError(t, wishlist.AddItem("prod-1"))
	require.NoError(t, wishlist.AddItem("prod-2"))
	err := wishlist.Clear()
	require.NoError(t, err)
	assert.Empty(t, wishlist.Items)
}

func TestWishlist_HasProduct(t *testing.T) {
	t.Parallel()

	wishlist, _ := NewWishlist("wl-1", "customer-1")
	assert.False(t, wishlist.HasProduct("prod-1"))

	require.NoError(t, wishlist.AddItem("prod-1"))
	assert.True(t, wishlist.HasProduct("prod-1"))
	assert.False(t, wishlist.HasProduct("prod-2"))
}

func TestWishlist_UpdatedAt_ChangesOnMutation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wishlist, _ := NewWishlist("wl-1", "customer-1")
		original := wishlist.UpdatedAt
		time.Sleep(time.Millisecond)
		require.NoError(t, wishlist.AddItem("prod-1"))
		assert.True(t, wishlist.UpdatedAt.After(original))
	})
}

func TestNewID(t *testing.T) {
	t.Parallel()
	ids := make(map[string]struct{}, 100)
	for range 100 {
		id := newID()
		require.NotEmpty(t, id)
		require.GreaterOrEqual(t, len(id), 26, "at least 128 bits of randomness")
		require.Regexp(t, `^[A-Z2-7]+$`, id)
		require.False(t, contains(ids, id), "duplicate id: %q", id)
		ids[id] = struct{}{}
	}
}

func contains(m map[string]struct{}, key string) bool {
	_, ok := m[key]
	return ok
}
