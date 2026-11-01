package contracts

import (
	"context"
	"testing"

	addressclient "minimart/internal/address/presentation/module/client"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	categoryclient "minimart/internal/category/presentation/module/client"
	customerclient "minimart/internal/customer/presentation/module/client"
	orderclient "minimart/internal/order/presentation/module/client"
	shippingclient "minimart/internal/shipping/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCatalog struct{}

func (stubCatalog) GetProduct(context.Context, string) (catalogclient.Product, error) {
	return catalogclient.Product{}, nil
}

func (stubCatalog) ReserveStock(context.Context, string, []catalogclient.LineItem) error {
	return nil
}

func (stubCatalog) ReleaseStock(context.Context, string) error { return nil }

type stubCategory struct{}

func (stubCategory) GetCategory(context.Context, string) (categoryclient.CategoryDTO, error) {
	return categoryclient.CategoryDTO{}, nil
}

type stubCustomer struct{}

func (stubCustomer) EnsureCustomer(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (stubCustomer) GetCustomerByID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

func (stubCustomer) GetCustomerByUserID(context.Context, string) (customerclient.Customer, error) {
	return customerclient.Customer{}, nil
}

type stubOrder struct{}

func (stubOrder) PlaceOrder(context.Context, orderclient.PlaceOrderRequest) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

func (stubOrder) GetOrder(context.Context, string) (orderclient.Order, error) {
	return orderclient.Order{}, nil
}

type stubShipping struct{ shippingclient.ShippingClient }

type stubAddress struct{ addressclient.AddressClient }

type stubAuth struct{}

func (stubAuth) GetAccountEmail(context.Context, string) (string, error) {
	return "test@example.com", nil
}

func TestVerify(t *testing.T) {
	t.Run("fails when catalog slot is empty", func(t *testing.T) {
		c := &Contracts{}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "catalog not registered")
	})

	t.Run("fails when category slot is empty", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CustomerClient: stubCustomer{}, OrderClient: stubOrder{}}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "category not registered")
	})

	t.Run("fails when order slot is empty", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CategoryClient: stubCategory{}, CustomerClient: stubCustomer{}}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "order not registered")
	})

	t.Run("fails when shipping slot is empty", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CategoryClient: stubCategory{}, CustomerClient: stubCustomer{}, OrderClient: stubOrder{}}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "shipping not registered")
	})

	t.Run("fails when address slot is empty", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CategoryClient: stubCategory{}, CustomerClient: stubCustomer{}, OrderClient: stubOrder{}, ShippingClient: stubShipping{}}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "address not registered")
	})

	t.Run("fails when auth slot is empty", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CategoryClient: stubCategory{}, CustomerClient: stubCustomer{}, OrderClient: stubOrder{}, ShippingClient: stubShipping{}, AddressClient: stubAddress{}}
		err := c.Verify()
		require.Error(t, err)
		assert.ErrorContains(t, err, "auth not registered")
	})

	t.Run("passes when every slot is filled", func(t *testing.T) {
		c := &Contracts{CatalogClient: stubCatalog{}, CategoryClient: stubCategory{}, CustomerClient: stubCustomer{}, OrderClient: stubOrder{}, ShippingClient: stubShipping{}, AddressClient: stubAddress{}, AuthClient: stubAuth{}}
		assert.NoError(t, c.Verify())
	})
}
