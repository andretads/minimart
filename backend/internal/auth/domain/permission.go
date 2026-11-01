package domain

import "time"

type Permission struct {
	ID        string
	CreatedAt time.Time
}

func NewPermission(id string) (Permission, error) {
	if id == "" {
		return Permission{}, ErrPermissionNotFound
	}
	return Permission{
		ID:        id,
		CreatedAt: time.Now(),
	}, nil
}
