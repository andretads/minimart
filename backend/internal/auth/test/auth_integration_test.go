//go:build integration

package test

import (
	"context"
	"fmt"
	"minimart/internal/auth/application"
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/postgres"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthIntegration(t *testing.T) {
	readPool, writePool := mustPools(t)
	repo := postgres.NewRepository(readPool, writePool)

	// Create a fake customer provisioner that uses the real customer module
	// For integration test, we'll use a simple mock
	prov := &fakeCustomerProvisioner{}

	svc := application.NewService(repo, prov, metrics.Noop{})

	t.Run("signup creates account", func(t *testing.T) {
		acc, err := svc.SignUp(t.Context(), application.SignUpRequest{
			Username: "testuser",
			Password: "password123",
			Email:    "testuser@example.com",
		})
		if err != nil {
			t.Fatalf("SignUp: %v", err)
		}
		if acc.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if acc.Username != "testuser" {
			t.Fatalf("expected testuser, got %s", acc.Username)
		}
		if !acc.PasswordMatches("password123") {
			t.Fatal("password should match")
		}

		got, err := svc.GetAccount(t.Context(), acc.ID)
		if err != nil {
			t.Fatalf("GetAccount: %v", err)
		}
		if got.ID != acc.ID {
			t.Fatalf("ID mismatch: %s vs %s", got.ID, acc.ID)
		}
	})

	t.Run("signup rejects duplicate username", func(t *testing.T) {
		_, err := svc.SignUp(t.Context(), application.SignUpRequest{
			Username: "testuser",
			Password: "different123",
			Email:    "different@example.com",
		})
		if err == nil {
			t.Fatal("expected ErrAccountExists")
		}
	})

	t.Run("authenticate validates credentials", func(t *testing.T) {
		acc, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: "testuser",
			Password: "password123",
		})
		if err != nil {
			t.Fatalf("Authenticate: %v", err)
		}
		if acc.ID == "" {
			t.Fatal("expected non-empty ID")
		}

		// Wrong password
		_, err = svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: "testuser",
			Password: "wrong",
		})
		if err == nil {
			t.Fatal("expected ErrInvalidCredentials")
		}
	})

	t.Run("unknown username returns invalid credentials (no enumeration)", func(t *testing.T) {
		_, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: "nobody",
			Password: "password123",
		})
		if err == nil {
			t.Fatal("expected ErrInvalidCredentials")
		}
	})
}

func TestChangePasswordIntegration(t *testing.T) {
	readPool, writePool := mustPools(t)
	repo := postgres.NewRepository(readPool, writePool)
	svc := application.NewService(repo, &fakeCustomerProvisioner{}, metrics.Noop{})

	username := fmt.Sprintf("pw-it-%d", time.Now().UnixNano())
	acc, err := svc.SignUp(t.Context(), application.SignUpRequest{
		Username: username,
		Password: "password123",
		Email:    username + "@example.com",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		_, _ = writePool.Exec(context.Background(), "DELETE FROM accounts WHERE id = $1", acc.ID)
	})

	t.Run("updates the stored hash", func(t *testing.T) {
		err := svc.ChangePassword(t.Context(), application.ChangePasswordRequest{
			AccountID:          acc.ID,
			CurrentPassword:    "password123",
			NewPassword:        "new-password",
			ConfirmNewPassword: "new-password",
		})
		require.NoError(t, err)

		authenticated, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: username,
			Password: "new-password",
		})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, authenticated.ID)

		_, err = svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: username,
			Password: "password123",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("missing account returns not found", func(t *testing.T) {
		err := repo.UpdatePassword(t.Context(), domain.Account{ID: "missing"})
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}

// TestReviewsWritePermissionSeeding proves the migration that introduces the
// reviews:write permission is applied: the permission exists, the customer and
// admin roles hold it, and a customer account can therefore submit a product
// review.
func TestReviewsWritePermissionSeeding(t *testing.T) {
	readPool, writePool := mustPools(t)
	repo := postgres.NewRepository(readPool, writePool)
	svc := application.NewService(repo, &fakeCustomerProvisioner{}, metrics.Noop{})

	perm, err := repo.GetPermission(t.Context(), "reviews:write")
	require.NoError(t, err)
	assert.Equal(t, "reviews:write", perm.ID)

	t.Run("customer and admin roles hold it", func(t *testing.T) {
		rows, err := readPool.Query(t.Context(), `
			SELECT role_id FROM role_permission WHERE permission_id = 'reviews:write' ORDER BY role_id
		`)
		require.NoError(t, err)
		defer rows.Close()

		var roles []string
		for rows.Next() {
			var roleID string
			require.NoError(t, rows.Scan(&roleID))
			roles = append(roles, roleID)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"admin", "customer"}, roles)
	})

	t.Run("customer role can submit reviews", func(t *testing.T) {
		username := fmt.Sprintf("rev-it-%d", time.Now().UnixNano())
		acc, err := svc.SignUp(t.Context(), application.SignUpRequest{
			Username: username,
			Password: "password123",
			Email:    username + "@example.com",
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			//nolint:usetesting // t.Context() is cancelled before cleanup runs
			_, _ = writePool.Exec(context.Background(), "DELETE FROM accounts WHERE id = $1", acc.ID)
		})

		require.NoError(t, svc.AssignDefaultRole(t.Context(), acc.ID))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, perms, "reviews:write")
		assert.Contains(t, perms, "catalog:read")
	})
}

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	readPool, err := newPool(ctx, testDSN)
	if err != nil {
		t.Fatalf("read pool: %v", err)
	}
	writePool, err := newPool(ctx, testDSN)
	if err != nil {
		t.Fatalf("write pool: %v", err)
	}
	return readPool, writePool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}

type fakeCustomerProvisioner struct{}

func (f *fakeCustomerProvisioner) EnsureCustomer(ctx context.Context, userID string) (client.Customer, error) {
	return client.Customer{ID: "cust-" + userID, UserID: userID}, nil
}
