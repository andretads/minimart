package domain

import (
	"time"

	"github.com/google/uuid"
)

type ShippingMethod struct {
	ID               string
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewShippingMethod(name, description string, active bool, estimatedMinDays, estimatedMaxDays int) (ShippingMethod, error) {
	if name == "" {
		return ShippingMethod{}, ErrInvalidInput
	}
	if estimatedMinDays < 0 || estimatedMaxDays < 0 || estimatedMinDays > estimatedMaxDays {
		return ShippingMethod{}, ErrInvalidTransition
	}
	now := time.Now().UTC()
	return ShippingMethod{
		ID:               uuid.New().String(),
		Name:             name,
		Description:      description,
		Active:           active,
		EstimatedMinDays: estimatedMinDays,
		EstimatedMaxDays: estimatedMaxDays,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (m *ShippingMethod) Update(name, description string, active bool, estimatedMinDays, estimatedMaxDays int) error {
	if name == "" {
		return ErrInvalidInput
	}
	if estimatedMinDays < 0 || estimatedMaxDays < 0 || estimatedMinDays > estimatedMaxDays {
		return ErrInvalidTransition
	}
	m.Name = name
	m.Description = description
	m.Active = active
	m.EstimatedMinDays = estimatedMinDays
	m.EstimatedMaxDays = estimatedMaxDays
	m.UpdatedAt = time.Now().UTC()
	return nil
}
