package memory

import (
	"context"
	"minimart/internal/category/domain"
	"slices"
	"strings"
	"sync"
	"time"
)

type Repository struct {
	mu         sync.RWMutex
	categories map[string]domain.Category
	byName     map[string]domain.Category
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		categories: map[string]domain.Category{},
		byName:     map[string]domain.Category{},
	}
}

func (r *Repository) GetByID(_ context.Context, id string) (domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.categories[id]
	if !ok {
		return domain.Category{}, domain.ErrCategoryNotFound
	}
	return c, nil
}

func (r *Repository) List(_ context.Context) ([]domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Category, 0, len(r.categories))
	for _, c := range r.categories {
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b domain.Category) int {
		if a.Name != b.Name {
			return strings.Compare(a.Name, b.Name)
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (r *Repository) Save(_ context.Context, c domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	lowerName := strings.ToLower(c.Name)
	if _, ok := r.byName[lowerName]; ok {
		return domain.ErrCategoryExists
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if c.Version == 0 {
		c.Version = 1
	}
	r.categories[c.ID] = c
	r.byName[lowerName] = c
	return nil
}

func (r *Repository) Update(_ context.Context, c domain.Category) (domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.categories[c.ID]
	if !ok {
		return domain.Category{}, domain.ErrCategoryNotFound
	}
	if stored.Version != c.Version {
		return domain.Category{}, domain.ErrCategoryOptimisticLock
	}
	if stored.Name != c.Name {
		lowerName := strings.ToLower(c.Name)
		if _, ok := r.byName[lowerName]; ok {
			return domain.Category{}, domain.ErrCategoryExists
		}
		delete(r.byName, strings.ToLower(stored.Name))
		r.byName[lowerName] = c
	}
	c.Version = stored.Version + 1
	c.CreatedAt = stored.CreatedAt
	c.UpdatedAt = time.Now().UTC()
	r.categories[c.ID] = c
	return c, nil
}

func (r *Repository) Delete(_ context.Context, id string, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.categories[id]
	if !ok {
		return domain.ErrCategoryNotFound
	}
	if stored.Version != version {
		return domain.ErrCategoryOptimisticLock
	}
	delete(r.byName, strings.ToLower(stored.Name))
	delete(r.categories, id)
	return nil
}
