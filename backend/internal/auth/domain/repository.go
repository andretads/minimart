package domain

import "context"

type Repository interface {
	GetByID(ctx context.Context, id string) (Account, error)
	GetByUsername(ctx context.Context, username string) (Account, error)
	Save(ctx context.Context, a Account) error
	UpdatePassword(ctx context.Context, a Account) error

	// RBAC methods
	GetRole(ctx context.Context, id string) (Role, error)
	GetPermission(ctx context.Context, id string) (Permission, error)
	GetRolesForAccount(ctx context.Context, accountID string) ([]Role, error)
	GetPermissionsForAccount(ctx context.Context, accountID string) ([]string, error)
	GetAccountsByUsername(ctx context.Context, usernames []string) ([]string, error)
	AssignRole(ctx context.Context, accountID, roleID string) error
	CreateRole(ctx context.Context, r Role) error
	CreatePermission(ctx context.Context, p Permission) error
	AddRolePermission(ctx context.Context, roleID, permissionID string) error
}
