package domain

import (
	countrycode "minimart/pkg/country"
	"time"

	"github.com/google/uuid"
)

type ShippingZone struct {
	ID              string
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewShippingZone(name, country, state, city, postalCodeStart, postalCodeEnd string) (ShippingZone, error) {
	if name == "" || country == "" {
		return ShippingZone{}, ErrZoneNotFound
	}
	now := time.Now().UTC()
	return ShippingZone{
		ID:              uuid.New().String(),
		Name:            name,
		Country:         country,
		State:           state,
		City:            city,
		PostalCodeStart: postalCodeStart,
		PostalCodeEnd:   postalCodeEnd,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (z *ShippingZone) Update(name, country, state, city, postalCodeStart, postalCodeEnd string) error {
	if name == "" || country == "" {
		return ErrZoneNotFound
	}
	z.Name = name
	z.Country = country
	z.State = state
	z.City = city
	z.PostalCodeStart = postalCodeStart
	z.PostalCodeEnd = postalCodeEnd
	z.UpdatedAt = time.Now().UTC()
	return nil
}

func (z ShippingZone) MatchesDestination(country, state, city, postalCode string) bool {
	if z.Country != "" && !countrycode.Equal(z.Country, country) {
		return false
	}
	if z.State != "" && z.State != state {
		return false
	}
	if z.City != "" && z.City != city {
		return false
	}
	if z.PostalCodeStart != "" && z.PostalCodeEnd != "" && postalCode != "" {
		if postalCode < z.PostalCodeStart || postalCode > z.PostalCodeEnd {
			return false
		}
	}
	return true
}
