package domain

import (
	"strings"
	"time"
)

const (
	maxNameLength = 100
	maxIconLength = 500
)

// Category is a category record.
// CreatedAt is stamped once when the category is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every persisted write.
type Category struct {
	ID        string
	Name      string
	Icon      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int
}

// NewCategory validates the inputs and returns a Category, or a domain error.
// name must be non-empty and max 100 characters (trimmed).
// icon is optional but max 500 characters (trimmed).
func NewCategory(id, name, icon string) (Category, error) {
	name, err := validateName(name)
	if err != nil {
		return Category{}, err
	}
	icon, err = validateIcon(icon)
	if err != nil {
		return Category{}, err
	}
	now := time.Now().UTC()
	return Category{
		ID:        id,
		Name:      name,
		Icon:      icon,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}, nil
}

// UpdateNameAndIcon updates the category's name and icon in place.
// The name is trimmed and validated (non-empty, max 100 chars).
// The icon is trimmed and validated (max 500 chars).
// UpdatedAt is set to now. Version is left untouched — the storage layer bumps it on persist.
func (c *Category) UpdateNameAndIcon(name, icon string) error {
	name, err := validateName(name)
	if err != nil {
		return err
	}
	icon, err = validateIcon(icon)
	if err != nil {
		return err
	}
	c.Name = name
	c.Icon = icon
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// validateName trims surrounding space and rejects an empty result or too long name.
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}
	if len(name) > maxNameLength {
		return "", ErrNameTooLong
	}
	return name, nil
}

// validateIcon trims surrounding space and rejects too long icon.
func validateIcon(icon string) (string, error) {
	icon = strings.TrimSpace(icon)
	if len(icon) > maxIconLength {
		return "", ErrIconTooLong
	}
	return icon, nil
}
