package postgres

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/postgres/reader"
	"minimart/internal/customer/infrastructure/postgres/writer"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerRepository implements domain.Repository using Postgres.
// It holds the write pool for mutations and the sqlc-generated reader/writer
// query objects for the read and write pools respectively.
type CustomerRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
}

var _ domain.Repository = (*CustomerRepository)(nil)

// NewCustomerRepository builds the repository with the provided read/write pools.
func NewCustomerRepository(readPool, writePool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
	}
}

// GetByID reads a customer by ID from the read pool.
// Returns ErrCustomerNotFound when no row exists.
func (r *CustomerRepository) GetByID(ctx context.Context, id string) (domain.Customer, error) {
	row, err := r.reader.GetCustomerByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, err
	}
	return toDomain(row), nil
}

// GetByUserID reads a customer by user ID from the read pool.
// Returns ErrCustomerNotFound when no row exists.
func (r *CustomerRepository) GetByUserID(ctx context.Context, userID string) (domain.Customer, error) {
	row, err := r.reader.GetCustomerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, err
	}
	return toDomainFromUser(row), nil
}

// Save inserts a new customer using the write pool. The insert uses
// ON CONFLICT (user_id) DO NOTHING so a duplicate user_id results in
// zero rows affected, which is mapped to ErrCustomerExists.
func (r *CustomerRepository) Save(ctx context.Context, c domain.Customer) error {
	rowsAffected, err := r.writer.CreateCustomer(ctx, writer.CreateCustomerParams{
		ID:          c.ID,
		UserID:      c.UserID,
		DisplayName: c.DisplayName,
		CreatedAt:   pgtype.Timestamptz{Time: c.CreatedAt, Valid: true},
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrCustomerExists
	}
	return nil
}

// UpdateProfile updates the customer's display_name on the write pool under
// optimistic lock: the expected version is carried on the entity's Version
// field. It returns the persisted customer with the new version and updated_at.
// Stale version reports ErrCustomerOptimisticLock; missing id reports
// ErrCustomerNotFound.
func (r *CustomerRepository) UpdateProfile(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	row, err := r.writer.UpdateCustomerProfile(ctx, writer.UpdateCustomerProfileParams{
		ID:          c.ID,
		DisplayName: c.DisplayName,
		Version:     int32(c.Version), //nolint:gosec // version is positive by construction
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.classifyNoMatch(ctx, c.ID)
		}
		return domain.Customer{}, err
	}
	c.Version = int(row.Version)
	c.UpdatedAt = row.UpdatedAt.Time
	return c, nil
}

// classifyNoMatch disambiguates a 0-row UPDATE: missing row vs stale version.
// The probe runs on the write pool to avoid replica lag masking a live row.
func (r *CustomerRepository) classifyNoMatch(ctx context.Context, id string) (domain.Customer, error) {
	exists, err := r.writer.CustomerExists(ctx, id)
	if err != nil {
		return domain.Customer{}, err
	}
	if exists {
		return domain.Customer{}, domain.ErrCustomerOptimisticLock
	}
	return domain.Customer{}, domain.ErrCustomerNotFound
}

// Deactivate marks the customer as deactivated on the write pool.
// It uses COALESCE(deactivated_at, NOW()) so the first deactivation timestamp
// is preserved on idempotent re-deactivation, and bumps the version.
// Returns the persisted customer with the new version, updated_at, and deactivated_at.
// Missing id reports ErrCustomerNotFound.
func (r *CustomerRepository) Deactivate(ctx context.Context, c domain.Customer) (domain.Customer, error) {
	row, err := r.writer.DeactivateCustomer(ctx, c.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Customer{}, domain.ErrCustomerNotFound
		}
		return domain.Customer{}, err
	}
	c.Version = int(row.Version)
	c.UpdatedAt = row.UpdatedAt.Time
	if row.DeactivatedAt.Valid {
		c.DeactivatedAt = &row.DeactivatedAt.Time
	}
	c.Status = domain.StatusDeactivated
	return c, nil
}

// toDomain converts a reader row (GetCustomerByID) to a domain.Customer.
// It handles the nullable deactivated_at column.
func toDomain(row reader.GetCustomerByIDRow) domain.Customer {
	var deactivatedAt *time.Time
	if row.DeactivatedAt.Valid {
		deactivatedAt = &row.DeactivatedAt.Time
	}
	return domain.Customer{
		ID:            row.ID,
		UserID:        row.UserID,
		DisplayName:   row.DisplayName,
		Status:        domain.Status(row.Status),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		Version:       int(row.Version),
		DeactivatedAt: deactivatedAt,
	}
}

// toDomainFromUser converts a reader row (GetCustomerByUserID) to a domain.Customer.
// It handles the nullable deactivated_at column.
func toDomainFromUser(row reader.GetCustomerByUserIDRow) domain.Customer {
	var deactivatedAt *time.Time
	if row.DeactivatedAt.Valid {
		deactivatedAt = &row.DeactivatedAt.Time
	}
	return domain.Customer{
		ID:            row.ID,
		UserID:        row.UserID,
		DisplayName:   row.DisplayName,
		Status:        domain.Status(row.Status),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		Version:       int(row.Version),
		DeactivatedAt: deactivatedAt,
	}
}
