package postgres

import (
	"context"
	"errors"
	"minimart/internal/category/domain"
	"minimart/internal/category/infrastructure/postgres/reader"
	"minimart/internal/category/infrastructure/postgres/writer"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CategoryRepository implements domain.Repository using Postgres.
type CategoryRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
}

var _ domain.Repository = (*CategoryRepository)(nil)

// NewCategoryRepository builds the repository with the provided read/write pools.
func NewCategoryRepository(readPool, writePool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
	}
}

// GetByID reads a category by ID from the read pool.
// Returns ErrCategoryNotFound when no row exists.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (domain.Category, error) {
	row, err := r.reader.GetCategory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Category{}, domain.ErrCategoryNotFound
		}
		return domain.Category{}, err
	}
	return toDomain(row), nil
}

// List reads all categories from the read pool, ordered by lower(name), id.
func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.reader.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

// Save inserts a new category using the write pool. The insert uses
// ON CONFLICT (LOWER(name)) DO NOTHING so a duplicate name (case-insensitive)
// results in zero rows affected, which is mapped to ErrCategoryExists.
func (r *CategoryRepository) Save(ctx context.Context, c domain.Category) error {
	rowsAffected, err := r.writer.CreateCategory(ctx, writer.CreateCategoryParams{
		ID:        c.ID,
		Name:      c.Name,
		Icon:      c.Icon,
		Version:   int32(c.Version), //nolint:gosec // version is positive by construction
		CreatedAt: pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrCategoryExists
	}
	return nil
}

// Update updates the category's name and icon on the write pool under
// optimistic lock: the expected version is carried on the entity's Version
// field. It returns the persisted category with the new version and updated_at.
// Stale version reports ErrCategoryOptimisticLock; missing id reports
// ErrCategoryNotFound.
func (r *CategoryRepository) Update(ctx context.Context, c domain.Category) (domain.Category, error) {
	row, err := r.writer.UpdateCategory(ctx, writer.UpdateCategoryParams{
		ID:      c.ID,
		Name:    c.Name,
		Icon:    c.Icon,
		Version: int32(c.Version), //nolint:gosec // version is positive by construction
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.classifyNoMatch(ctx, c.ID)
		}
		return domain.Category{}, err
	}
	c.Version = int(row.Version)
	c.UpdatedAt = row.UpdatedAt.Time
	return c, nil
}

// Delete deletes a category by ID under optimistic lock on the write pool.
// It reports ErrCategoryOptimisticLock if the version does not match,
// and ErrCategoryNotFound if the category does not exist.
func (r *CategoryRepository) Delete(ctx context.Context, id string, version int) error {
	rowsAffected, err := r.writer.DeleteCategory(ctx, writer.DeleteCategoryParams{
		ID:      id,
		Version: int32(version), //nolint:gosec // version is positive by construction
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return r.classifyNoMatchDelete(ctx, id)
	}
	return nil
}

// classifyNoMatch disambiguates a 0-row UPDATE: missing row vs stale version.
// The probe runs on the write pool to avoid replica lag masking a live row.
func (r *CategoryRepository) classifyNoMatch(ctx context.Context, id string) (domain.Category, error) {
	exists, err := r.writer.CategoryExists(ctx, id)
	if err != nil {
		return domain.Category{}, err
	}
	if exists {
		return domain.Category{}, domain.ErrCategoryOptimisticLock
	}
	return domain.Category{}, domain.ErrCategoryNotFound
}

// classifyNoMatchDelete disambiguates a 0-row DELETE: missing row vs stale version.
func (r *CategoryRepository) classifyNoMatchDelete(ctx context.Context, id string) error {
	exists, err := r.writer.CategoryExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrCategoryOptimisticLock
	}
	return domain.ErrCategoryNotFound
}

// toDomain converts a reader row to a domain.Category.
func toDomain(row reader.Category) domain.Category {
	return domain.Category{
		ID:        row.ID,
		Name:      row.Name,
		Icon:      row.Icon,
		Version:   int(row.Version),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
