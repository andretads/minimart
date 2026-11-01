package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewShippingRate(t *testing.T) {
	tests := []struct {
		name               string
		zoneID, methodID   string
		baseCostCents      int64
		freeThresholdCents int64
		active             bool
		expectErr          bool
		errIs              error
	}{
		{"valid", "zone1", "method1", 599, 5000, true, false, nil},
		{"free shipping", "zone1", "method1", 599, 10000, true, false, nil},
		{"zero cost", "zone1", "method1", 0, 0, true, false, nil},
		{"negative base cost", "zone1", "method1", -1, 0, true, true, ErrInvalidTransition},
		{"negative free threshold", "zone1", "method1", 599, -1, true, true, ErrInvalidTransition},
		{"empty zone", "", "method1", 599, 5000, true, true, ErrRateNotFound},
		{"empty method", "zone1", "", 599, 5000, true, true, ErrRateNotFound},
		{"inactive rate", "zone1", "method1", 599, 5000, false, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewShippingRate(tt.zoneID, tt.methodID, tt.baseCostCents, tt.freeThresholdCents, tt.active)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.zoneID, r.ZoneID)
			assert.Equal(t, tt.methodID, r.MethodID)
			assert.Equal(t, tt.baseCostCents, r.BaseCostCents)
			assert.Equal(t, tt.freeThresholdCents, r.FreeThresholdCents)
			assert.Equal(t, tt.active, r.Active)
			assert.False(t, r.CreatedAt.IsZero())
			assert.Equal(t, r.CreatedAt, r.UpdatedAt)
		})
	}
}

func TestShippingRate_Update(t *testing.T) {
	r, _ := NewShippingRate("zone1", "method1", 599, 5000, true)
	originalCreated := r.CreatedAt

	err := r.Update(1299, 10000, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1299), r.BaseCostCents)
	assert.Equal(t, int64(10000), r.FreeThresholdCents)
	assert.False(t, r.Active)
	assert.Equal(t, originalCreated, r.CreatedAt)
	assert.True(t, r.UpdatedAt.After(originalCreated))
}

func TestShippingRate_Update_Invalid(t *testing.T) {
	r, _ := NewShippingRate("zone1", "method1", 599, 5000, true)

	tests := []struct {
		name          string
		baseCostCents int64
		freeThreshold int64
		errIs         error
	}{
		{"negative base cost", -1, 0, ErrInvalidTransition},
		{"negative free threshold", 599, -1, ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Update(tt.baseCostCents, tt.freeThreshold, true)
			require.ErrorIs(t, err, tt.errIs)
		})
	}
}

func TestShippingRate_CalculateCost(t *testing.T) {
	r, _ := NewShippingRate("zone1", "method1", 599, 5000, true)

	assert.Equal(t, int64(599), r.CalculateCost(1000))
	assert.Equal(t, int64(0), r.CalculateCost(5000))
	assert.Equal(t, int64(0), r.CalculateCost(10000))

	// Inactive rate still calculates cost (caller should check active status)
	r2, _ := NewShippingRate("zone1", "method2", 999, 0, false)
	assert.Equal(t, int64(999), r2.CalculateCost(1000))
}
