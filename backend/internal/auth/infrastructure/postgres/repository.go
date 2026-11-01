package postgres

import (
	"context"
	"errors"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/postgres/reader"
	"minimart/internal/auth/infrastructure/postgres/writer"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
}

var _ domain.Repository = (*Repository)(nil)

func NewRepository(readPool, writePool *pgxpool.Pool) *Repository {
	return &Repository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
	}
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Account, error) {
	row, err := r.reader.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toDomainFromID(row), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (domain.Account, error) {
	row, err := r.reader.GetAccountByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Account{}, domain.ErrAccountNotFound
		}
		return domain.Account{}, err
	}
	return toDomainFromUsername(row), nil
}

func (r *Repository) Save(ctx context.Context, acc domain.Account) error {
	rowsAffected, err := r.writer.CreateAccount(ctx, writer.CreateAccountParams{
		ID:           acc.ID,
		Username:     acc.Username,
		Email:        pgtype.Text{String: acc.Email, Valid: true},
		PasswordHash: acc.PasswordHash.Bytes(),
		CreatedAt:    pgtype.Timestamptz{Time: acc.CreatedAt, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccountExists
		}
		// Check for unique violation on email
		if err.Error() != "" && (contains(err.Error(), "email") && (contains(err.Error(), "duplicate") || contains(err.Error(), "unique"))) {
			return domain.ErrEmailExists
		}
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAccountExists
	}
	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, acc domain.Account) error {
	rowsAffected, err := r.writer.UpdatePassword(ctx, writer.UpdatePasswordParams{
		ID:           acc.ID,
		PasswordHash: acc.PasswordHash.Bytes(),
		UpdatedAt:    pgtype.Timestamptz{Time: acc.UpdatedAt, Valid: true},
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAccountNotFound
	}
	return nil
}

func (r *Repository) GetRole(ctx context.Context, id string) (domain.Role, error) {
	row, err := r.reader.GetRole(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Role{}, domain.ErrRoleNotFound
		}
		return domain.Role{}, err
	}
	return domain.Role{
		ID:          row.ID,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt.Time,
	}, nil
}

func (r *Repository) GetPermission(ctx context.Context, id string) (domain.Permission, error) {
	row, err := r.reader.GetPermission(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Permission{}, domain.ErrPermissionNotFound
		}
		return domain.Permission{}, err
	}
	return domain.Permission{
		ID:        row.ID,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *Repository) GetRolesForAccount(ctx context.Context, accountID string) ([]domain.Role, error) {
	rows, err := r.reader.GetRolesForAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	roles := make([]domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = domain.Role{
			ID:          row.ID,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return roles, nil
}

func (r *Repository) GetPermissionsForAccount(ctx context.Context, accountID string) ([]string, error) {
	return r.reader.GetPermissionsForAccount(ctx, accountID)
}

func (r *Repository) GetAccountsByUsername(ctx context.Context, usernames []string) ([]string, error) {
	if len(usernames) == 0 {
		return nil, nil
	}
	return r.reader.GetAccountsByUsername(ctx, usernames)
}

func (r *Repository) AssignRole(ctx context.Context, accountID, roleID string) error {
	return r.writer.AssignRole(ctx, writer.AssignRoleParams{
		AccountID: accountID,
		RoleID:    roleID,
	})
}

func (r *Repository) CreateRole(ctx context.Context, role domain.Role) error {
	return r.writer.CreateRole(ctx, writer.CreateRoleParams{
		ID:          role.ID,
		Description: pgtype.Text{String: role.Description, Valid: role.Description != ""},
		CreatedAt:   pgtype.Timestamptz{Time: role.CreatedAt, Valid: !role.CreatedAt.IsZero()},
	})
}

func (r *Repository) CreatePermission(ctx context.Context, perm domain.Permission) error {
	return r.writer.CreatePermission(ctx, writer.CreatePermissionParams{
		ID:        perm.ID,
		CreatedAt: pgtype.Timestamptz{Time: perm.CreatedAt, Valid: !perm.CreatedAt.IsZero()},
	})
}

func (r *Repository) AddRolePermission(ctx context.Context, roleID, permissionID string) error {
	return r.writer.AddRolePermission(ctx, writer.AddRolePermissionParams{
		RoleID:       roleID,
		PermissionID: permissionID,
	})
}

func toDomainFromID(row reader.GetAccountByIDRow) domain.Account {
	return domain.LoadAccount(
		row.ID,
		row.Username,
		row.Email.String,
		row.PasswordHash,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}

func toDomainFromUsername(row reader.GetAccountByUsernameRow) domain.Account {
	return domain.LoadAccount(
		row.ID,
		row.Username,
		row.Email.String,
		row.PasswordHash,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
