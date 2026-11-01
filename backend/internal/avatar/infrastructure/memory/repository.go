package memory

import (
	"context"
	"minimart/internal/avatar/domain"
	"sync"
	"time"
)

type Repository struct {
	mu      sync.RWMutex
	avatars map[string]domain.Avatar
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		avatars: map[string]domain.Avatar{},
	}
}

func (r *Repository) GetByAccountID(_ context.Context, accountID string) (domain.Avatar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.avatars[accountID]
	if !ok {
		return domain.Avatar{}, domain.ErrAvatarNotFound
	}
	return a, nil
}

func (r *Repository) Save(_ context.Context, a domain.Avatar) (domain.Avatar, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.avatars[a.AccountID]
	if ok {
		a.Version = existing.Version + 1
		a.CreatedAt = existing.CreatedAt
		a.UpdatedAt = time.Now().UTC()
	} else {
		if a.CreatedAt.IsZero() {
			a.CreatedAt = time.Now().UTC()
		}
		if a.UpdatedAt.IsZero() {
			a.UpdatedAt = time.Now().UTC()
		}
		if a.Version == 0 {
			a.Version = 1
		}
	}
	r.avatars[a.AccountID] = a
	return a, nil
}
