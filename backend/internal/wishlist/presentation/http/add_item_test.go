package wishlisthttp

import (
	"bytes"
	"minimart/internal/wishlist/application"
	"minimart/pkg/metrics"
	"minimart/pkg/response"
	"net/http"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_AddItem(t *testing.T) {
	t.Run("adds item successfully", func(t *testing.T) {
		mux := newTestMux(t)
		body := bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusOK, rec.Code)
		var r response.Response
		decodeResponse(t, rec, &r)
		assert.True(t, r.Success)
	})

	t.Run("duplicate product returns 409", func(t *testing.T) {
		mux := newTestMux(t)
		// First add
		body := bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		require.Equal(t, http.StatusOK, rec.Code)

		// Second add (duplicate)
		body = bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec = doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("product not found returns 404", func(t *testing.T) {
		mux := newTestMux(t)
		body := bytes.NewBufferString(`{"product_id":"prod-999"}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux := newTestMux(t)
		body := bytes.NewBufferString(`{"product_id":"prod-1"}`)
		rec := doRequestUnauth(t, mux, http.MethodPost, "/v1/wishlist/items", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("missing product_id returns 400", func(t *testing.T) {
		mux := newTestMux(t)
		body := bytes.NewBufferString(`{}`)
		rec := doRequest(t, mux, http.MethodPost, "/v1/wishlist/items?user_id=user-1", body.Bytes(), map[string]string{"Content-Type": "application/json"})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestService_AddItem_Direct(t *testing.T) {
	svc := newAddItemTestService(t)
	_, err := svc.AddItem(t.Context(), application.AddItemRequest{UserID: "user-1", ProductID: "prod-999"})
	require.Error(t, err)
	assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
}

func newAddItemTestService(t *testing.T) *application.Service {
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
