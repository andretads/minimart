//go:build integration

package test

import (
	"context"
	"minimart/internal/avatar/domain"
	"minimart/internal/avatar/infrastructure/postgres"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedTestAccount creates a test account in the database.
func seedTestAccount(ctx context.Context, pool *pgxpool.Pool, accountID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO accounts (id, username, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, accountID, "testuser-"+accountID, []byte("hash"))
	return err
}

// TestAvatarRepository_SaveGetByAccountID round-trips an avatar through the real Postgres
// adapter: Save, then GetByAccountID, and asserts the value comes back unchanged.
func TestAvatarRepository_SaveGetByAccountID(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	accountID := "it-" + uniqueAccountID()
	if seederr := seedTestAccount(ctx, pool, accountID); seederr != nil {
		t.Fatalf("seed test account: %v", seederr)
	}
	defer cleanupAccount(ctx, pool, accountID)

	repo := postgres.NewAvatarRepository(pool, pool)

	want, err := domain.NewAvatar(accountID, "avatars/"+accountID+"/avatar", "minimart", "image/jpeg", 1024)
	if err != nil {
		t.Fatalf("NewAvatar: %v", err)
	}
	if _, err = repo.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByAccountID(ctx, accountID)
	if err != nil {
		t.Fatalf("GetByAccountID: %v", err)
	}
	if got.AccountID != want.AccountID || got.ObjectKey != want.ObjectKey ||
		got.Bucket != want.Bucket || got.ContentType != want.ContentType ||
		got.SizeBytes != want.SizeBytes {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
	if got.Version != 1 {
		t.Fatalf("new avatar version = %d, want 1", got.Version)
	}
	if got.CreatedAt.Sub(want.CreatedAt).Abs() > time.Millisecond {
		t.Fatalf("created_at mismatch: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if got.UpdatedAt.Sub(want.UpdatedAt).Abs() > time.Millisecond {
		t.Fatalf("updated_at mismatch: got %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}

	// Re-saving the same account_id is an upsert: created_at must survive while
	// the other columns refresh, matching the in-memory adapter.
	resave := want
	resave.ObjectKey = "avatars/" + accountID + "/avatar-v2"
	resave.ContentType = "image/png"
	resave.SizeBytes = 2048
	resave.CreatedAt = want.CreatedAt.Add(time.Hour)
	resave.UpdatedAt = want.UpdatedAt.Add(time.Hour)
	if _, err = repo.Save(ctx, resave); err != nil {
		t.Fatalf("Save (upsert): %v", err)
	}
	got, err = repo.GetByAccountID(ctx, accountID)
	if err != nil {
		t.Fatalf("GetByAccountID after upsert: %v", err)
	}
	if got.CreatedAt.Sub(want.CreatedAt).Abs() > time.Millisecond {
		t.Fatalf("created_at was rewritten on upsert: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if got.ObjectKey != resave.ObjectKey || got.ContentType != resave.ContentType || got.SizeBytes != resave.SizeBytes {
		t.Fatalf("upsert did not refresh columns: got %+v", got)
	}
	if got.UpdatedAt.Before(want.UpdatedAt) {
		t.Fatalf("updated_at moved backwards on upsert: got %v, want >= %v", got.UpdatedAt, want.UpdatedAt)
	}
	if got.Version != 2 {
		t.Fatalf("upsert version = %d, want 2", got.Version)
	}
}

// TestAvatarRepository_GetByAccountIDNotFound returns ErrAvatarNotFound for missing account.
func TestAvatarRepository_GetByAccountIDNotFound(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewAvatarRepository(pool, pool)

	_, err = repo.GetByAccountID(ctx, "nonexistent-account")
	if err == nil {
		t.Fatal("expected ErrAvatarNotFound, got nil")
	}
}

// cleanupAccount deletes the test account from the database.
func cleanupAccount(ctx context.Context, pool *pgxpool.Pool, accountID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, accountID)
}

// uniqueAccountID generates a unique account ID for tests.
func uniqueAccountID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
