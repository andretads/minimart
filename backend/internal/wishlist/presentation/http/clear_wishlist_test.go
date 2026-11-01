package wishlisthttp

import (
	"bytes"
	"minimart/internal/wishlist/application"
	"minimart/pkg/metrics"
	"net/http"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_ClearWishlist(t *testing.T) {
	t.Run("clears wishlist", func(t *testing.T) {
		mux := newTestMux(t)
		// First add items
		body := bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		body = bytes.NewBufferString(`{"product_id":"prod-2"}`)
		rec = doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Clear wishlist
		rec = doRequest(t, mux, http.MethodDelete, "/v1/wishlist?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux := newTestMux(t)
		rec := doRequestUnauth(t, mux, http.MethodDelete, "/v1/wishlist", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestService_ClearWishlist_Direct(t *testing.T) {
	svc := newClearWishlistTestService(t)
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "user-1", ProductID: "prod-1"})
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "user-1", ProductID: "prod-2"})

	err := svc.ClearWishlist(t.Context(), application.ClearWishlistRequest{UserID: "user-1"})
	require.NoError(t, err)

	wishlist, err := svc.GetWishlist(t.Context(), application.GetWishlistRequest{UserID: "user-1"})
	require.NoError(t, err)
	assert.Empty(t, wishlist.Items)
}

func newClearWishlistTestService(t *testing.T) *application.Service {
	t.Helper()
	repo := newFakeRepo()
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
		"prod-2": {ID: "prod-2", Name: "Product 2", PriceCents: 2000},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	return application.NewService(repo, mod, metrics.Noop{})
}
