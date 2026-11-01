package domain

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestID() string {
	return rand.Text()
}

func TestNewAccount(t *testing.T) {
	t.Run("creates a valid account", func(t *testing.T) {
		id := newTestID()
		username := "testuser"
		password := "password123"
		email := "test@example.com"

		acc, err := NewAccount(id, username, password, email)
		require.NoError(t, err)
		assert.Equal(t, id, acc.ID)
		assert.Equal(t, username, acc.Username)
		assert.Equal(t, email, acc.Email)
		assert.True(t, acc.PasswordMatches(password))
		assert.False(t, acc.CreatedAt.IsZero())
		assert.False(t, acc.UpdatedAt.IsZero())
		assert.Equal(t, acc.CreatedAt, acc.UpdatedAt)
	})

	t.Run("rejects empty username", func(t *testing.T) {
		_, err := NewAccount("acc-1", "", "password123", "test@example.com")
		assert.ErrorIs(t, err, ErrEmptyUsername)
	})

	t.Run("rejects whitespace username", func(t *testing.T) {
		_, err := NewAccount("acc-1", "   ", "password123", "test@example.com")
		assert.ErrorIs(t, err, ErrEmptyUsername)
	})

	t.Run("rejects long username", func(t *testing.T) {
		long := strings.Repeat("a", 201)
		_, err := NewAccount("acc-1", long, "password123", "test@example.com")
		assert.ErrorIs(t, err, ErrUsernameTooLong)
	})

	t.Run("rejects empty password", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "", "test@example.com")
		assert.ErrorIs(t, err, ErrEmptyPassword)
	})

	t.Run("rejects short password", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "12345", "test@example.com")
		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("rejects long password", func(t *testing.T) {
		long := strings.Repeat("a", 73)
		_, err := NewAccount("acc-1", "testuser", long, "test@example.com")
		assert.ErrorIs(t, err, ErrPasswordTooLong)
	})

	t.Run("trims username whitespace", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "  testuser  ", "password123", "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, "testuser", acc.Username)
	})

	t.Run("rejects empty email", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "")
		assert.ErrorIs(t, err, ErrEmptyEmail)
	})

	t.Run("rejects whitespace email", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "   ")
		assert.ErrorIs(t, err, ErrEmptyEmail)
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "invalid-email")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("rejects email without domain", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "test@")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("rejects email without local part", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "@example.com")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("rejects email with spaces", func(t *testing.T) {
		_, err := NewAccount("acc-1", "testuser", "password123", "test @example.com")
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("rejects long email", func(t *testing.T) {
		long := strings.Repeat("a", 250) + "@example.com"
		_, err := NewAccount("acc-1", "testuser", "password123", long)
		assert.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("accepts valid email formats", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.org",
			"user+tag@example.co.uk",
			"user123@test-domain.com",
		}
		for _, e := range validEmails {
			acc, err := NewAccount("acc-"+newTestID(), "user", "password123", e)
			require.NoError(t, err, "email: %s", e)
			assert.Equal(t, e, acc.Email)
		}
	})
}

func TestLoadAccount(t *testing.T) {
	t.Run("loads account from stored values", func(t *testing.T) {
		id := newTestID()
		username := "testuser"
		email := "test@example.com"
		createdAt := time.Now().UTC().Add(-time.Hour)
		updatedAt := time.Now().UTC()

		acc := LoadAccount(id, username, email, []byte{}, createdAt, updatedAt)
		assert.Equal(t, id, acc.ID)
		assert.Equal(t, username, acc.Username)
		assert.Equal(t, email, acc.Email)
		assert.Equal(t, createdAt, acc.CreatedAt)
		assert.Equal(t, updatedAt, acc.UpdatedAt)
	})
}

func TestAccount_ChangePassword(t *testing.T) {
	t.Run("rehashes the password and keeps identity", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		require.NoError(t, err)
		acc.UpdatedAt = time.Now().Add(-time.Hour)

		updated, err := acc.ChangePassword("password123", "new-password")

		require.NoError(t, err)
		assert.Equal(t, acc.ID, updated.ID)
		assert.Equal(t, acc.Username, updated.Username)
		assert.Equal(t, acc.Email, updated.Email)
		assert.True(t, updated.PasswordMatches("new-password"))
		assert.False(t, updated.PasswordMatches("password123"))
		assert.True(t, updated.UpdatedAt.After(acc.UpdatedAt))
	})

	t.Run("rejects an incorrect current password", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		require.NoError(t, err)

		_, err = acc.ChangePassword("wrong", "new-password")

		require.ErrorIs(t, err, ErrInvalidCurrentPassword)
		assert.True(t, acc.PasswordMatches("password123"))
	})

	t.Run("rejects an empty new password", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		require.NoError(t, err)

		_, err = acc.ChangePassword("password123", "")

		assert.ErrorIs(t, err, ErrEmptyPassword)
	})

	t.Run("rejects a short new password", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		require.NoError(t, err)

		_, err = acc.ChangePassword("password123", "12345")

		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})

	t.Run("rejects a long new password", func(t *testing.T) {
		acc, err := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		require.NoError(t, err)

		_, err = acc.ChangePassword("password123", strings.Repeat("a", 73))

		assert.ErrorIs(t, err, ErrPasswordTooLong)
	})
}

func TestAccount_PasswordMatches(t *testing.T) {
	t.Run("returns true for correct password", func(t *testing.T) {
		acc, _ := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		assert.True(t, acc.PasswordMatches("password123"))
	})

	t.Run("returns false for incorrect password", func(t *testing.T) {
		acc, _ := NewAccount("acc-1", "testuser", "password123", "test@example.com")
		assert.False(t, acc.PasswordMatches("wrong"))
	})

	t.Run("returns false for empty password on loaded account", func(t *testing.T) {
		acc := LoadAccount("acc-1", "testuser", "test@example.com", []byte{}, time.Now(), time.Now())
		assert.False(t, acc.PasswordMatches("anything"))
	})
}
