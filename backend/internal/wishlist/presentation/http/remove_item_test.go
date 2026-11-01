package wishlisthttp

import (
	"bytes"
	"minimart/internal/wishlist/application"
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
	"minimart/pkg/response"
	"net/http"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_RemoveItem(t *testing.T) {
	t.Run("removes existing item", func(t *testing.T) {
		mux := newTestMux(t)
		// First add item
		body := bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Remove item
		rec = doRequest(t, mux, http.MethodDelete, "/v1/wishlist/items/prod-1?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		var r response.Response
		decodeResponse(t, rec, &r)
		assert.True(t, r.Success)
	})

	t.Run("non-existent product returns 404", func(t *testing.T) {
		mux := newTestMux(t)
		rec := doRequest(t, mux, http.MethodDelete, "/v1/wishlist/items/prod-999?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux := newTestMux(t)
		rec := doRequestUnauth(t, mux, http.MethodDelete, "/v1/wishlist/items/prod-1", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestService_RemoveItem_Direct(t *testing.T) {
	svc := newRemoveItemTestService(t)
	// First add item to wishlist
	_, _ = svc.AddItem(t.Context(), application.AddItemRequest{UserID: "user-1", ProductID: "prod-1"})

	// Then try to remove non-existent product
	_, err := svc.RemoveItem(t.Context(), application.RemoveItemRequest{UserID: "user-1", ProductID: "prod-999"})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrProductNotInWishlist)
}

func newRemoveItemTestService(t *testing.T) *application.Service {
	t.Helper()
	repo := newFakeRepo()
	catalog := &fakeCatalog{products: map[string]catalogclient.Product{
		"prod-1": {ID: "prod-1", Name: "Product 1", PriceCents: 1000},
	}}
	customer := newFakeCustomer()
	order := newFakeOrder()
	category := &fakeCategory{categories: map[string]categoryclient.CategoryDTO{
		"cat-1": {ID: "cat-1", Name: "Test Category"},
	}}
	mod := &testModules{fakeCatalog: catalog, fakeCustomer: customer, fakeOrder: order, fakeCategory: category}
	return application.NewService(repo, mod, metrics.Noop{})
}
