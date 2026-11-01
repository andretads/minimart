package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShippingZone(t *testing.T) {
	tests := []struct {
		name        string
		nameVal     string
		country     string
		state       string
		city        string
		postalStart string
		postalEnd   string
		expectErr   bool
		errIs       error
	}{
		{"valid country only", "Brazil", "BR", "", "", "", "", false, nil},
		{"valid with state", "SP", "BR", "SP", "", "", "", false, nil},
		{"valid with city", "Sao Paulo", "BR", "SP", "Sao Paulo", "", "", false, nil},
		{"valid with postal range", "CEP Range", "BR", "SP", "Sao Paulo", "01000-000", "09999-999", false, nil},
		{"empty name", "", "BR", "", "", "", "", true, ErrZoneNotFound},
		{"empty country", "Zone", "", "", "", "", "", true, ErrZoneNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z, err := NewShippingZone(tt.nameVal, tt.country, tt.state, tt.city, tt.postalStart, tt.postalEnd)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.nameVal, z.Name)
			assert.Equal(t, tt.country, z.Country)
			assert.Equal(t, tt.state, z.State)
			assert.Equal(t, tt.city, z.City)
			assert.Equal(t, tt.postalStart, z.PostalCodeStart)
			assert.Equal(t, tt.postalEnd, z.PostalCodeEnd)
			assert.False(t, z.CreatedAt.IsZero())
			assert.Equal(t, z.CreatedAt, z.UpdatedAt)
		})
	}
}

func TestShippingZone_Update(t *testing.T) {
	z, _ := NewShippingZone("Brazil", "BR", "SP", "Sao Paulo", "01000-000", "09999-999")
	originalCreated := z.CreatedAt

	err := z.Update("Brazil Updated", "BR", "RJ", "Rio", "20000-000", "29999-999")
	require.NoError(t, err)
	assert.Equal(t, "Brazil Updated", z.Name)
	assert.Equal(t, "BR", z.Country)
	assert.Equal(t, "RJ", z.State)
	assert.Equal(t, "Rio", z.City)
	assert.Equal(t, "20000-000", z.PostalCodeStart)
	assert.Equal(t, "29999-999", z.PostalCodeEnd)
	assert.Equal(t, originalCreated, z.CreatedAt)
	assert.True(t, z.UpdatedAt.After(originalCreated))
}

func TestShippingZone_Update_Invalid(t *testing.T) {
	z, _ := NewShippingZone("Brazil", "BR", "", "", "", "")

	tests := []struct {
		name    string
		nameVal string
		country string
		errIs   error
	}{
		{"empty name", "", "BR", ErrZoneNotFound},
		{"empty country", "Zone", "", ErrZoneNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := z.Update(tt.nameVal, tt.country, "", "", "", "")
			require.ErrorIs(t, err, tt.errIs)
		})
	}
}

func TestShippingZone_MatchesDestination(t *testing.T) {
	z, _ := NewShippingZone("Brazil", "BR", "", "", "", "")

	tests := []struct {
		name     string
		country  string
		state    string
		city     string
		postal   string
		expected bool
	}{
		{"match country only", "BR", "RJ", "Rio", "20000-000", true},
		{"wrong country", "US", "SP", "Sao Paulo", "05000-000", false},
		{"different state same country", "BR", "RJ", "Sao Paulo", "05000-000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, z.MatchesDestination(tt.country, tt.state, tt.city, tt.postal))
		})
	}
}

func TestShippingZone_MatchesDestination_WithPostalRange(t *testing.T) {
	z, _ := NewShippingZone("Brazil", "BR", "SP", "Sao Paulo", "01000-000", "09999-999")

	tests := []struct {
		name     string
		country  string
		state    string
		city     string
		postal   string
		expected bool
	}{
		{"match all", "BR", "SP", "Sao Paulo", "05000-000", true},
		{"wrong country", "US", "SP", "Sao Paulo", "05000-000", false},
		{"postal in range", "BR", "SP", "Sao Paulo", "01000-000", true},
		{"postal out of range", "BR", "SP", "Sao Paulo", "10000-000", false},
		{"postal at start boundary", "BR", "SP", "Sao Paulo", "01000-000", true},
		{"postal at end boundary", "BR", "SP", "Sao Paulo", "09999-999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, z.MatchesDestination(tt.country, tt.state, tt.city, tt.postal))
		})
	}
}

func TestShippingZone_MatchesDestination_WithStateAndCity(t *testing.T) {
	z, _ := NewShippingZone("Sao Paulo", "BR", "SP", "Sao Paulo", "", "")

	tests := []struct {
		name     string
		country  string
		state    string
		city     string
		postal   string
		expected bool
	}{
		{"match all", "BR", "SP", "Sao Paulo", "05000-000", true},
		{"wrong country", "US", "SP", "Sao Paulo", "05000-000", false},
		{"wrong state", "BR", "RJ", "Sao Paulo", "05000-000", false},
		{"wrong city", "BR", "SP", "Rio", "05000-000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, z.MatchesDestination(tt.country, tt.state, tt.city, tt.postal))
		})
	}
}

func TestShippingZone_MatchesDestination_NormalizesCountry(t *testing.T) {
	z, _ := NewShippingZone("Brazil", "BR", "", "", "", "")
	assert.True(t, z.MatchesDestination("Brasil", "RJ", "Rio", "20000-000"))
	assert.True(t, z.MatchesDestination("br", "RJ", "Rio", "20000-000"))
	assert.False(t, z.MatchesDestination("US", "RJ", "Rio", "20000-000"))
}
