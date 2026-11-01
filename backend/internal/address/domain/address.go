package domain

import (
	"errors"
	countrycode "minimart/pkg/country"
	"strings"
	"time"
)

const (
	maxLabelLength    = 50
	maxFullNameLength = 100
	maxStreetLength   = 200
	maxCityLength     = 100
	maxStateLength    = 100
	maxZipLength      = 20
	maxCountryLength  = 100
	maxPhoneLength    = 30
)

var (
	ErrAddressNotFound      = errors.New("address: endereço não encontrado")
	ErrAddressExists        = errors.New("address: endereço já existe")
	ErrInvalidVersion       = errors.New("address: versão inválida")
	ErrDefaultAddressExists = errors.New("address: cliente já possui um endereço padrão")
	ErrEmptyCustomerID      = errors.New("address: id do cliente é obrigatório")
	ErrEmptyLabel           = errors.New("address: rótulo é obrigatório")
	ErrLabelTooLong         = errors.New("address: rótulo excede o tamanho máximo")
	ErrEmptyFullName        = errors.New("address: nome completo é obrigatório")
	ErrFullNameTooLong      = errors.New("address: nome completo excede o tamanho máximo")
	ErrEmptyStreet          = errors.New("address: logradouro é obrigatório")
	ErrStreetTooLong        = errors.New("address: logradouro excede o tamanho máximo")
	ErrEmptyCity            = errors.New("address: cidade é obrigatória")
	ErrCityTooLong          = errors.New("address: cidade excede o tamanho máximo")
	ErrEmptyState           = errors.New("address: estado é obrigatório")
	ErrStateTooLong         = errors.New("address: estado excede o tamanho máximo")
	ErrEmptyZip             = errors.New("address: CEP é obrigatório")
	ErrZipTooLong           = errors.New("address: CEP excede o tamanho máximo")
	ErrEmptyCountry         = errors.New("address: país é obrigatório")
	ErrCountryTooLong       = errors.New("address: país excede o tamanho máximo")
	ErrPhoneTooLong         = errors.New("address: telefone excede o tamanho máximo")
)

// Address is an address record linked to a customer.
// Code outside tests should create Address values through NewAddress, so
// normal application flow is valid by construction.
//
// CreatedAt is stamped once when the address is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every persisted write. It is not set by domain logic.
type Address struct {
	ID         string
	CustomerID string
	IsDefault  bool
	Label      string
	FullName   string
	Street     string
	City       string
	State      string
	Zip        string
	Country    string
	Phone      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int
}

// AddressFields is the editable payload of an address: everything the
// customer supplies on create and on update.
type AddressFields struct {
	Label    string
	FullName string
	Street   string
	City     string
	State    string
	Zip      string
	Country  string
	Phone    string
}

// normalized returns a copy of the fields with every value trimmed, the
// country normalized to ISO 3166-1 alpha-2, and every length and required
// rule enforced. It returns the first violation found, in field order.
func (f AddressFields) normalized() (AddressFields, error) {
	out := AddressFields{
		Label:    strings.TrimSpace(f.Label),
		FullName: strings.TrimSpace(f.FullName),
		Street:   strings.TrimSpace(f.Street),
		City:     strings.TrimSpace(f.City),
		State:    strings.TrimSpace(f.State),
		Zip:      strings.TrimSpace(f.Zip),
		Phone:    strings.TrimSpace(f.Phone),
	}
	if out.Label == "" {
		return AddressFields{}, ErrEmptyLabel
	}
	if len(out.Label) > maxLabelLength {
		return AddressFields{}, ErrLabelTooLong
	}

	if out.FullName == "" {
		return AddressFields{}, ErrEmptyFullName
	}
	if len(out.FullName) > maxFullNameLength {
		return AddressFields{}, ErrFullNameTooLong
	}

	if out.Street == "" {
		return AddressFields{}, ErrEmptyStreet
	}
	if len(out.Street) > maxStreetLength {
		return AddressFields{}, ErrStreetTooLong
	}

	if out.City == "" {
		return AddressFields{}, ErrEmptyCity
	}
	if len(out.City) > maxCityLength {
		return AddressFields{}, ErrCityTooLong
	}

	if out.State == "" {
		return AddressFields{}, ErrEmptyState
	}
	if len(out.State) > maxStateLength {
		return AddressFields{}, ErrStateTooLong
	}

	if out.Zip == "" {
		return AddressFields{}, ErrEmptyZip
	}
	if len(out.Zip) > maxZipLength {
		return AddressFields{}, ErrZipTooLong
	}

	if strings.TrimSpace(f.Country) == "" {
		return AddressFields{}, ErrEmptyCountry
	}
	out.Country = countrycode.Normalize(strings.TrimSpace(f.Country))
	if len(out.Country) > maxCountryLength {
		return AddressFields{}, ErrCountryTooLong
	}

	if len(out.Phone) > maxPhoneLength {
		return AddressFields{}, ErrPhoneTooLong
	}

	return out, nil
}

// NewAddress validates the fields and returns an Address, or a domain error.
// customerID must be non-empty. All fields are trimmed and validated.
// The address starts with CreatedAt and UpdatedAt set to now, Version = 1.
func NewAddress(id, customerID string, fields AddressFields, isDefault bool) (Address, error) {
	if customerID == "" {
		return Address{}, ErrEmptyCustomerID
	}
	norm, err := fields.normalized()
	if err != nil {
		return Address{}, err
	}

	now := time.Now().UTC()
	return Address{
		ID:         id,
		CustomerID: customerID,
		IsDefault:  isDefault,
		Label:      norm.Label,
		FullName:   norm.FullName,
		Street:     norm.Street,
		City:       norm.City,
		State:      norm.State,
		Zip:        norm.Zip,
		Country:    norm.Country,
		Phone:      norm.Phone,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

// UpdateDetails replaces the address's mutable fields in place.
// All fields are trimmed and validated. UpdatedAt is set to now.
// Version is left untouched — the storage layer bumps it on persist.
// Returns the first validation error encountered, or nil on success.
func (a *Address) UpdateDetails(fields AddressFields) error {
	norm, err := fields.normalized()
	if err != nil {
		return err
	}
	a.Label = norm.Label
	a.FullName = norm.FullName
	a.Street = norm.Street
	a.City = norm.City
	a.State = norm.State
	a.Zip = norm.Zip
	a.Country = norm.Country
	a.Phone = norm.Phone
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// SetDefault marks the address as the default. UpdatedAt is set to now.
// Version is left untouched — the storage layer bumps it on persist.
func (a *Address) SetDefault() {
	if !a.IsDefault {
		a.IsDefault = true
		a.UpdatedAt = time.Now().UTC()
	}
}

// UnsetDefault marks the address as not default. UpdatedAt is set to now.
// Version is left untouched — the storage layer bumps it on persist.
func (a *Address) UnsetDefault() {
	if a.IsDefault {
		a.IsDefault = false
		a.UpdatedAt = time.Now().UTC()
	}
}

// IsDefaultAddress returns true if the address is the default.
func (a Address) IsDefaultAddress() bool {
	return a.IsDefault
}
