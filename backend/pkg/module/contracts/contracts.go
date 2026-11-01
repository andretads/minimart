package contracts

import (
	"errors"

	address "minimart/internal/address/presentation/module/client"
	auth "minimart/internal/auth/presentation/module/client"
	catalog "minimart/internal/catalog/presentation/module/client"
	category "minimart/internal/category/presentation/module/client"
	customer "minimart/internal/customer/presentation/module/client"
	order "minimart/internal/order/presentation/module/client"
	shipping "minimart/internal/shipping/presentation/module/client"
)

type Modules interface {
	catalog.CatalogClient
	category.CategoryClient
	customer.CustomerClient
	order.OrderClient
	shipping.ShippingClient
	address.AddressClient
}

type Contracts struct {
	catalog.CatalogClient
	category.CategoryClient
	customer.CustomerClient
	order.OrderClient
	shipping.ShippingClient
	address.AddressClient
	auth.AuthClient
}

func (c *Contracts) Verify() error {
	if c.CatalogClient == nil {
		return errors.New("contracts: catalog not registered")
	}
	if c.CategoryClient == nil {
		return errors.New("contracts: category not registered")
	}
	if c.CustomerClient == nil {
		return errors.New("contracts: customer not registered")
	}
	if c.OrderClient == nil {
		return errors.New("contracts: order not registered")
	}
	if c.ShippingClient == nil {
		return errors.New("contracts: shipping not registered")
	}
	if c.AddressClient == nil {
		return errors.New("contracts: address not registered")
	}
	if c.AuthClient == nil {
		return errors.New("contracts: auth not registered")
	}
	return nil
}
