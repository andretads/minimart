package client

import (
	"context"
	"errors"
)

var ErrAddressNotFound = errors.New("address: endereço não encontrado")

type Address struct {
	ID         string
	CustomerID string
	Label      string
	FullName   string
	Street     string
	City       string
	State      string
	Zip        string
	Country    string
	Phone      string
	IsDefault  bool
	Version    int
	CreatedAt  string
	UpdatedAt  string
}

type GetAddressRequest struct {
	ID         string
	CustomerID string
}

type AddressClient interface {
	GetAddress(ctx context.Context, req GetAddressRequest) (Address, error)
	GetDefaultAddressByCustomerID(ctx context.Context, customerID string) (Address, error)
}

type CreateAddressRequest struct {
	Label    string
	FullName string
	Street   string
	City     string
	State    string
	Zip      string
	Country  string
	Phone    string
}

type UpdateAddressRequest struct {
	Label    string
	FullName string
	Street   string
	City     string
	State    string
	Zip      string
	Country  string
	Phone    string
	Version  int
}
