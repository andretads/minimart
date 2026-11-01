package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShippingMethod(t *testing.T) {
	tests := []struct {
		name      string
		nameVal   string
		desc      string
		active    bool
		minDays   int
		maxDays   int
		expectErr bool
		errIs     error
	}{
		{"valid standard", "Standard Delivery", "Standard 3-5 days", true, 3, 5, false, nil},
		{"valid express", "Express Delivery", "Express 1-2 days", true, 1, 2, false, nil},
		{"empty name", "", "desc", true, 1, 2, true, ErrInvalidInput},
		{"negative min days", "Method", "desc", true, -1, 2, true, ErrInvalidTransition},
		{"min > max days", "Method", "desc", true, 5, 3, true, ErrInvalidTransition},
		{"inactive method", "Inactive", "desc", false, 1, 2, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewShippingMethod(tt.nameVal, tt.desc, tt.active, tt.minDays, tt.maxDays)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.nameVal, m.Name)
			assert.Equal(t, tt.active, m.Active)
			assert.Equal(t, tt.minDays, m.EstimatedMinDays)
			assert.Equal(t, tt.maxDays, m.EstimatedMaxDays)
			assert.False(t, m.CreatedAt.IsZero())
			assert.Equal(t, m.CreatedAt, m.UpdatedAt)
		})
	}
}

func TestShippingMethod_Update(t *testing.T) {
	m, _ := NewShippingMethod("Standard", "desc", true, 3, 5)
	originalCreated := m.CreatedAt

	err := m.Update("Express", "Fast delivery", false, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, "Express", m.Name)
	assert.Equal(t, "Fast delivery", m.Description)
	assert.False(t, m.Active)
	assert.Equal(t, 1, m.EstimatedMinDays)
	assert.Equal(t, 2, m.EstimatedMaxDays)
	assert.Equal(t, originalCreated, m.CreatedAt)
	assert.True(t, m.UpdatedAt.After(originalCreated))
}

func TestShippingMethod_Update_Invalid(t *testing.T) {
	m, _ := NewShippingMethod("Standard", "desc", true, 3, 5)

	tests := []struct {
		name    string
		nameVal string
		minDays int
		maxDays int
		errIs   error
	}{
		{"empty name", "", 1, 2, ErrInvalidInput},
		{"negative min", "Express", -1, 2, ErrInvalidTransition},
		{"min > max", "Express", 5, 3, ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.Update(tt.nameVal, "desc", true, tt.minDays, tt.maxDays)
			require.ErrorIs(t, err, tt.errIs)
		})
	}
}
