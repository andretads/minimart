package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPasswordHash(t *testing.T) {
	t.Run("creates valid hash", func(t *testing.T) {
		hash, err := newPasswordHash("my-password")
		require.NoError(t, err)
		require.NotEmpty(t, hash.value)

		err = bcrypt.CompareHashAndPassword(hash.value, []byte("my-password"))
		assert.NoError(t, err)
	})

	t.Run("creates different hashes for same password", func(t *testing.T) {
		hash1, _ := newPasswordHash("same")
		hash2, _ := newPasswordHash("same")
		assert.NotEqual(t, hash1.value, hash2.value)
	})
}

func TestLoadPasswordHash(t *testing.T) {
	t.Run("loads hash from bytes", func(t *testing.T) {
		hash := loadPasswordHash([]byte("pre-hashed"))
		assert.Equal(t, []byte("pre-hashed"), hash.value)
	})
}

func TestPasswordHash_Matches(t *testing.T) {
	t.Run("matches correct secret", func(t *testing.T) {
		hash, _ := newPasswordHash("secret")
		assert.True(t, hash.Matches("secret"))
	})

	t.Run("does not match wrong secret", func(t *testing.T) {
		hash, _ := newPasswordHash("secret")
		assert.False(t, hash.Matches("wrong"))
	})

	t.Run("returns false for empty hash", func(t *testing.T) {
		var h PasswordHash
		assert.False(t, h.Matches("any"))
	})

	t.Run("returns false for nil hash", func(t *testing.T) {
		var h *PasswordHash
		assert.False(t, h.Matches("any"))
	})
}
