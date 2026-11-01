package domain

import "time"

type Role struct {
	ID          string
	Description string
	CreatedAt   time.Time
}

func NewRole(id, description string) (Role, error) {
	if id == "" {
		return Role{}, ErrRoleNotFound
	}
	return Role{
		ID:          id,
		Description: description,
		CreatedAt:   time.Now(),
	}, nil
}
