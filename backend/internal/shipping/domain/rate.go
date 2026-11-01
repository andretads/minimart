package domain

import (
	"time"

	"github.com/google/uuid"
)

type ShippingRate struct {
	ID                 string
	ZoneID             string
	MethodID           string
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewShippingRate(zoneID, methodID string, baseCostCents, freeThresholdCents int64, active bool) (ShippingRate, error) {
	if zoneID == "" || methodID == "" {
		return ShippingRate{}, ErrRateNotFound
	}
	if baseCostCents < 0 || freeThresholdCents < 0 {
		return ShippingRate{}, ErrInvalidTransition
	}
	now := time.Now().UTC()
	return ShippingRate{
		ID:                 uuid.New().String(),
		ZoneID:             zoneID,
		MethodID:           methodID,
		BaseCostCents:      baseCostCents,
		FreeThresholdCents: freeThresholdCents,
		Active:             active,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func (r *ShippingRate) Update(baseCostCents, freeThresholdCents int64, active bool) error {
	if baseCostCents < 0 || freeThresholdCents < 0 {
		return ErrInvalidTransition
	}
	r.BaseCostCents = baseCostCents
	r.FreeThresholdCents = freeThresholdCents
	r.Active = active
	r.UpdatedAt = time.Now().UTC()
	return nil
}

func (r ShippingRate) CalculateCost(subtotalCents int64) int64 {
	if r.FreeThresholdCents > 0 && subtotalCents >= r.FreeThresholdCents {
		return 0
	}
	return r.BaseCostCents
}
