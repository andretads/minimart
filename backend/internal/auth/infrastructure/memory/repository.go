package memory

import (
	"context"
	"minimart/internal/auth/domain"
	"sync"
)

type Repository struct {
	mu          sync.RWMutex
	accounts    map[string]domain.Account
	byUsername  map[string]domain.Account
	byEmail     map[string]domain.Account
	roles       map[string]domain.Role
	permissions map[string]domain.Permission
	userRoles   map[string]map[string]bool // accountID -> roleID -> exists
	rolePerms   map[string]map[string]bool // roleID -> permissionID -> exists
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		accounts:    map[string]domain.Account{},
		byUsername:  map[string]domain.Account{},
		byEmail:     map[string]domain.Account{},
		roles:       map[string]domain.Role{},
		permissions: map[string]domain.Permission{},
		userRoles:   map[string]map[string]bool{},
		rolePerms:   map[string]map[string]bool{},
	}
}

func (r *Repository) GetByID(_ context.Context, id string) (domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acc, ok := r.accounts[id]
	if !ok {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	return acc, nil
}

func (r *Repository) GetByUsername(_ context.Context, username string) (domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acc, ok := r.byUsername[username]
	if !ok {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	return acc, nil
}

func (r *Repository) Save(_ context.Context, acc domain.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byUsername[acc.Username]; ok {
		return domain.ErrAccountExists
	}
	if acc.Email != "" {
		if _, ok := r.byEmail[acc.Email]; ok {
			return domain.ErrEmailExists
		}
	}
	r.accounts[acc.ID] = acc
	r.byUsername[acc.Username] = acc
	if acc.Email != "" {
		r.byEmail[acc.Email] = acc
	}
	return nil
}

func (r *Repository) UpdatePassword(_ context.Context, acc domain.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.accounts[acc.ID]; !ok {
		return domain.ErrAccountNotFound
	}
	r.accounts[acc.ID] = acc
	r.byUsername[acc.Username] = acc
	if acc.Email != "" {
		r.byEmail[acc.Email] = acc
	}
	return nil
}

// RBAC methods

func (r *Repository) GetRole(_ context.Context, id string) (domain.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, ok := r.roles[id]
	if !ok {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	return role, nil
}

func (r *Repository) GetPermission(_ context.Context, id string) (domain.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	perm, ok := r.permissions[id]
	if !ok {
		return domain.Permission{}, domain.ErrPermissionNotFound
	}
	return perm, nil
}

func (r *Repository) GetRolesForAccount(_ context.Context, accountID string) ([]domain.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	roleIDs := r.userRoles[accountID]
	if roleIDs == nil {
		return nil, nil
	}
	roles := make([]domain.Role, 0, len(roleIDs))
	for id := range roleIDs {
		if role, ok := r.roles[id]; ok {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

func (r *Repository) GetPermissionsForAccount(_ context.Context, accountID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	roleIDs := r.userRoles[accountID]
	if roleIDs == nil {
		return nil, nil
	}
	perms := make(map[string]struct{})
	for roleID := range roleIDs {
		permIDs := r.rolePerms[roleID]
		for permID := range permIDs {
			perms[permID] = struct{}{}
		}
	}
	result := make([]string, 0, len(perms))
	for id := range perms {
		result = append(result, id)
	}
	return result, nil
}

func (r *Repository) GetAccountsByUsername(_ context.Context, usernames []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(usernames) == 0 {
		return nil, nil
	}
	var ids []string
	for _, username := range usernames {
		if acc, ok := r.byUsername[username]; ok {
			ids = append(ids, acc.ID)
		}
	}
	return ids, nil
}

func (r *Repository) AssignRole(_ context.Context, accountID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.userRoles[accountID] == nil {
		r.userRoles[accountID] = map[string]bool{}
	}
	r.userRoles[accountID][roleID] = true
	return nil
}

func (r *Repository) CreateRole(_ context.Context, role domain.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[role.ID] = role
	return nil
}

func (r *Repository) CreatePermission(_ context.Context, perm domain.Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.permissions[perm.ID] = perm
	return nil
}

func (r *Repository) AddRolePermission(_ context.Context, roleID, permissionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rolePerms[roleID] == nil {
		r.rolePerms[roleID] = map[string]bool{}
	}
	r.rolePerms[roleID][permissionID] = true
	return nil
}
