package memory

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	repo := New()
	ctx := t.Context()

	// Save a customer
	c, err := domain.NewCustomer("c1", "user-1", "John")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, c))
	assert.Equal(t, 1, c.Version)

	// GetByID
	got, err := repo.GetByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "John", got.DisplayName)
	assert.Equal(t, domain.StatusActive, got.Status)
	assert.Equal(t, 1, got.Version)

	// GetByUserID
	got, err = repo.GetByUserID(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)

	// UpdateProfile bumps version and preserves created_at
	updated, err := repo.UpdateProfile(ctx, domain.Customer{
		ID:          c.ID,
		UserID:      c.UserID,
		DisplayName: "Jane",
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Version:     c.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, "Jane", updated.DisplayName)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, c.CreatedAt, updated.CreatedAt, "created_at must be preserved")
	assert.False(t, updated.UpdatedAt.Before(c.UpdatedAt), "updated_at must not move backwards")

	// Read back and verify
	got, err = repo.GetByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, "Jane", got.DisplayName)
	assert.Equal(t, 2, got.Version)
	assert.Equal(t, c.CreatedAt, got.CreatedAt)

	// UpdateProfile of unknown id returns ErrCustomerNotFound
	_, err = repo.UpdateProfile(ctx, domain.Customer{
		ID:      "unknown",
		Version: 1,
	})
	require.ErrorIs(t, err, domain.ErrCustomerNotFound)

	// Deactivate bumps version and preserves created_at, sets deactivated_at
	deact, err := repo.Deactivate(ctx, domain.Customer{
		ID:          updated.ID,
		UserID:      updated.UserID,
		DisplayName: updated.DisplayName,
		Status:      updated.Status,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
		Version:     updated.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDeactivated, deact.Status)
	assert.NotNil(t, deact.DeactivatedAt)
	assert.Equal(t, updated.Version+1, deact.Version)
	assert.Equal(t, updated.CreatedAt, deact.CreatedAt)
	assert.False(t, deact.UpdatedAt.Before(updated.UpdatedAt))

	// Read back and verify
	got, err = repo.GetByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDeactivated, got.Status)
	assert.NotNil(t, got.DeactivatedAt)
	assert.Equal(t, updated.Version+1, got.Version)
	assert.Equal(t, updated.CreatedAt, got.CreatedAt)

	// Second deactivate is idempotent on status/deactivated_at but bumps version
	deact2, err := repo.Deactivate(ctx, domain.Customer{
		ID:          deact.ID,
		UserID:      deact.UserID,
		DisplayName: deact.DisplayName,
		Status:      deact.Status,
		CreatedAt:   deact.CreatedAt,
		UpdatedAt:   deact.UpdatedAt,
		Version:     deact.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, deact.DeactivatedAt, deact2.DeactivatedAt)
	assert.Equal(t, deact.Version+1, deact2.Version)

	// Deactivate of unknown id returns ErrCustomerNotFound
	_, err = repo.Deactivate(ctx, domain.Customer{
		ID:      "unknown",
		Version: 1,
	})
	require.ErrorIs(t, err, domain.ErrCustomerNotFound)
}

func TestRepositoryOptimisticLock(t *testing.T) {
	repo := New()
	ctx := t.Context()

	// Create initial customer
	c, err := domain.NewCustomer("c1", "user-1", "John")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, c))
	assert.Equal(t, 1, c.Version)

	// Two writers read the same version-1 snapshot
	snapshot := c

	// First writer updates successfully
	updated1, err := repo.UpdateProfile(ctx, domain.Customer{
		ID:          snapshot.ID,
		UserID:      snapshot.UserID,
		DisplayName: "Jane",
		Status:      snapshot.Status,
		CreatedAt:   snapshot.CreatedAt,
		UpdatedAt:   snapshot.UpdatedAt,
		Version:     snapshot.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, "Jane", updated1.DisplayName)
	assert.Equal(t, 2, updated1.Version)

	// Second writer uses the stale version and gets rejected
	_, err = repo.UpdateProfile(ctx, domain.Customer{
		ID:          snapshot.ID,
		UserID:      snapshot.UserID,
		DisplayName: "Jane2",
		Status:      snapshot.Status,
		CreatedAt:   snapshot.CreatedAt,
		UpdatedAt:   snapshot.UpdatedAt,
		Version:     snapshot.Version, // stale!
	})
	require.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)

	// First writer's change is intact
	got, err := repo.GetByID(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, "Jane", got.DisplayName)
	assert.Equal(t, 2, got.Version)
}
