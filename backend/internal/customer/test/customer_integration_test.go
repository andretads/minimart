//go:build integration

package test

import (
	"context"
	"minimart/internal/customer/application"
	"minimart/internal/customer/infrastructure/postgres"
	"minimart/pkg/metrics"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCustomerIntegration(t *testing.T) {
	readPool, writePool := mustPools(t)
	repo := postgres.NewCustomerRepository(readPool, writePool)
	svc := application.NewService(repo, metrics.Noop{})

	t.Run("create and get customer by ID", func(t *testing.T) {
		created, err := svc.CreateCustomer(t.Context(), application.CreateCustomerRequest{
			UserID:      "user-1",
			DisplayName: "John",
		})
		if err != nil {
			t.Fatalf("CreateCustomer: %v", err)
		}
		if created.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if created.UserID != "user-1" {
			t.Fatalf("expected user-1, got %s", created.UserID)
		}
		if created.DisplayName != "John" {
			t.Fatalf("expected John, got %s", created.DisplayName)
		}
		if created.Status != "active" {
			t.Fatalf("expected active, got %s", created.Status)
		}

		got, err := svc.GetCustomerByID(t.Context(), created.ID)
		if err != nil {
			t.Fatalf("GetCustomerByID: %v", err)
		}
		if got.ID != created.ID {
			t.Fatalf("ID mismatch: %s vs %s", got.ID, created.ID)
		}
	})

	t.Run("get customer by user ID", func(t *testing.T) {
		got, err := svc.GetCustomerByUserID(t.Context(), "user-1")
		if err != nil {
			t.Fatalf("GetCustomerByUserID: %v", err)
		}
		if got.UserID != "user-1" {
			t.Fatalf("expected user-1, got %s", got.UserID)
		}
	})

	t.Run("duplicate user_id returns ErrCustomerExists", func(t *testing.T) {
		_, err := svc.CreateCustomer(t.Context(), application.CreateCustomerRequest{
			UserID: "user-1",
		})
		if err == nil {
			t.Fatal("expected ErrCustomerExists")
		}
	})

	t.Run("update customer profile", func(t *testing.T) {
		created, err := svc.CreateCustomer(t.Context(), application.CreateCustomerRequest{
			UserID:      "user-2",
			DisplayName: "Alice",
		})
		if err != nil {
			t.Fatalf("CreateCustomer: %v", err)
		}

		updated, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Alice Updated",
			Version:     created.Version,
		})
		if err != nil {
			t.Fatalf("UpdateCustomerProfile: %v", err)
		}
		if updated.DisplayName != "Alice Updated" {
			t.Fatalf("expected Alice Updated, got %s", updated.DisplayName)
		}

		got, err := svc.GetCustomerByID(t.Context(), created.ID)
		if err != nil {
			t.Fatalf("GetCustomerByID: %v", err)
		}
		if got.DisplayName != "Alice Updated" {
			t.Fatalf("expected Alice Updated, got %s", got.DisplayName)
		}
	})

	t.Run("deactivate customer is idempotent", func(t *testing.T) {
		created, err := svc.CreateCustomer(t.Context(), application.CreateCustomerRequest{
			UserID: "user-3",
		})
		if err != nil {
			t.Fatalf("CreateCustomer: %v", err)
		}

		deact1, err := svc.DeactivateCustomer(t.Context(), created.ID)
		if err != nil {
			t.Fatalf("DeactivateCustomer: %v", err)
		}
		if deact1.Status != "deactivated" {
			t.Fatalf("expected deactivated, got %s", deact1.Status)
		}

		deact2, err := svc.DeactivateCustomer(t.Context(), created.ID)
		if err != nil {
			t.Fatalf("DeactivateCustomer second call: %v", err)
		}
		if deact2.Status != "deactivated" {
			t.Fatalf("expected deactivated, got %s", deact2.Status)
		}
	})

	t.Run("get non-existent returns ErrCustomerNotFound", func(t *testing.T) {
		_, err := svc.GetCustomerByID(t.Context(), "non-existent")
		if err == nil {
			t.Fatal("expected ErrCustomerNotFound")
		}
	})

	t.Run("EnsureCustomer is idempotent", func(t *testing.T) {
		c1, err := svc.EnsureCustomer(t.Context(), "ensure-user")
		if err != nil {
			t.Fatalf("EnsureCustomer: %v", err)
		}
		c2, err := svc.EnsureCustomer(t.Context(), "ensure-user")
		if err != nil {
			t.Fatalf("EnsureCustomer second call: %v", err)
		}
		if c1.ID != c2.ID {
			t.Fatalf("EnsureCustomer returned different IDs: %s vs %s", c1.ID, c2.ID)
		}
	})
}

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	// Using the same DSN for both read and write since test container is single instance
	// In production these would be separate pools.
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
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
