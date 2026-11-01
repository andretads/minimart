package domain

import (
	"strings"
	"time"
)

const maxDisplayNameLength = 200

// Status is a customer's lifecycle state.
type Status string

const (
	// StatusActive means the customer record is active and can be used.
	StatusActive Status = "active"
	// StatusDeactivated means the customer record has been deactivated.
	// Deactivation is idempotent and preserves the original deactivation timestamp.
	StatusDeactivated Status = "deactivated"
)

// Customer is a customer record linked to a user identity.
// Code outside tests should create Customer values through NewCustomer, so
// normal application flow is valid by construction.
//
// CreatedAt is stamped once when the customer is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every persisted write. It is not set by domain logic.
// DeactivatedAt is nil until the customer is deactivated, then holds the
// timestamp of the first deactivation (preserved on idempotent re-deactivation).
type Customer struct {
	ID            string
	UserID        string
	DisplayName   string
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Version       int
	DeactivatedAt *time.Time
}

// NewCustomer validates the inputs and returns a Customer, or a domain error.
// userID must be non-empty. displayName is trimmed and must not exceed 200 characters.
// The customer starts active with CreatedAt and UpdatedAt set to now, Version = 1.
func NewCustomer(id, userID, displayName string) (Customer, error) {
	if userID == "" {
		return Customer{}, ErrEmptyUserID
	}
	customer := Customer{
		ID:          id,
		UserID:      userID,
		DisplayName: strings.TrimSpace(displayName),
		Status:      StatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Version:     1,
	}
	if len(customer.DisplayName) > maxDisplayNameLength {
		return Customer{}, ErrDisplayNameTooLong
	}
	return customer, nil
}

// IsActive returns true if the customer's status is active.
func (c Customer) IsActive() bool {
	return c.Status == StatusActive
}

// UpdateDisplayName updates the customer's display name in place.
// The name is trimmed and validated (max 200 chars). UpdatedAt is set to now.
// Version is left untouched — the storage layer bumps it on persist.
// Returns ErrDisplayNameTooLong if the name exceeds the limit.
func (c *Customer) UpdateDisplayName(displayName string) error {
	c.DisplayName = strings.TrimSpace(displayName)
	if len(c.DisplayName) > maxDisplayNameLength {
		return ErrDisplayNameTooLong
	}
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate marks the customer as deactivated. It is idempotent: if already
// deactivated, it returns without changing anything, preserving the original
// DeactivatedAt timestamp. UpdatedAt is set to now on the first call.
// Version is left untouched — the storage layer bumps it on persist.
func (c *Customer) Deactivate() {
	if c.Status == StatusDeactivated {
		return
	}
	now := time.Now().UTC()
	c.Status = StatusDeactivated
	c.DeactivatedAt = &now
	c.UpdatedAt = now
}
