package memory

import (
	"minimart/internal/avatar/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()
	accountID := "test-account-1"
	validAvatar := func() domain.Avatar {
		a, _ := domain.NewAvatar(accountID, "avatars/test/avatar", "bucket", "image/jpeg", 1024)
		return a
	}

	t.Run("GetByAccountID returns ErrAvatarNotFound when missing", func(t *testing.T) {
		_, err := repo.GetByAccountID(ctx, "missing")
		assert.ErrorIs(t, err, domain.ErrAvatarNotFound)
	})

	t.Run("Save inserts new avatar with version 1", func(t *testing.T) {
		a := validAvatar()
		saved, err := repo.Save(ctx, a)
		require.NoError(t, err)
		assert.Equal(t, 1, saved.Version)
		assert.False(t, saved.CreatedAt.IsZero())
		assert.False(t, saved.UpdatedAt.IsZero())
		assert.Equal(t, a.AccountID, saved.AccountID)
	})

	t.Run("GetByAccountID returns saved avatar", func(t *testing.T) {
		got, err := repo.GetByAccountID(ctx, accountID)
		require.NoError(t, err)
		assert.Equal(t, accountID, got.AccountID)
	})

	t.Run("Save replaces existing avatar and increments version", func(t *testing.T) {
		original, _ := repo.GetByAccountID(ctx, accountID)
		newAvatar := validAvatar()
		newAvatar.ObjectKey = "avatars/test/avatar-new"
		newAvatar.ContentType = "image/png"
		newAvatar.SizeBytes = 2048

		saved, err := repo.Save(ctx, newAvatar)
		require.NoError(t, err)
		assert.Equal(t, original.Version+1, saved.Version)
		assert.Equal(t, original.CreatedAt, saved.CreatedAt)
		assert.True(t, saved.UpdatedAt.After(original.UpdatedAt))
		assert.Equal(t, "avatars/test/avatar-new", saved.ObjectKey)
		assert.Equal(t, "image/png", saved.ContentType)
		assert.Equal(t, int64(2048), saved.SizeBytes)
	})

	t.Run("Save preserves CreatedAt on update", func(t *testing.T) {
		original, _ := repo.GetByAccountID(ctx, accountID)
		newAvatar := validAvatar()
		newAvatar.ObjectKey = "avatars/test/avatar-v3"
		newAvatar.ContentType = "image/webp"
		newAvatar.SizeBytes = 4096
		newAvatar.CreatedAt = original.CreatedAt.Add(time.Hour)

		saved, err := repo.Save(ctx, newAvatar)
		require.NoError(t, err)
		assert.Equal(t, original.CreatedAt, saved.CreatedAt)
	})
}
