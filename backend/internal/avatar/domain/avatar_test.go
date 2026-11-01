package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAvatar(t *testing.T) {
	now := time.Now().UTC()
	validAccountID := "A1B2C3D4E5F6G7H8I9J0K1L2M3"
	validObjectKey := "avatars/test/avatar"
	validBucket := "minimart"
	validContentType := "image/jpeg"
	validSizeBytes := int64(1024)

	t.Run("creates valid avatar", func(t *testing.T) {
		a, err := NewAvatar(validAccountID, validObjectKey, validBucket, validContentType, validSizeBytes)
		require.NoError(t, err)
		assert.Equal(t, validAccountID, a.AccountID)
		assert.Equal(t, validObjectKey, a.ObjectKey)
		assert.Equal(t, validBucket, a.Bucket)
		assert.Equal(t, validContentType, a.ContentType)
		assert.Equal(t, validSizeBytes, a.SizeBytes)
		assert.Equal(t, 1, a.Version)
		assert.False(t, a.CreatedAt.IsZero())
		assert.False(t, a.UpdatedAt.IsZero())
		assert.True(t, a.CreatedAt.After(now.Add(-time.Second)))
		assert.True(t, a.UpdatedAt.After(now.Add(-time.Second)))
	})

	t.Run("trims account id", func(t *testing.T) {
		a, err := NewAvatar("  "+validAccountID+"  ", validObjectKey, validBucket, validContentType, validSizeBytes)
		require.NoError(t, err)
		assert.Equal(t, validAccountID, a.AccountID)
	})

	t.Run("trims object key", func(t *testing.T) {
		a, err := NewAvatar(validAccountID, "  "+validObjectKey+"  ", validBucket, validContentType, validSizeBytes)
		require.NoError(t, err)
		assert.Equal(t, validObjectKey, a.ObjectKey)
	})

	t.Run("trims bucket", func(t *testing.T) {
		a, err := NewAvatar(validAccountID, validObjectKey, "  "+validBucket+"  ", validContentType, validSizeBytes)
		require.NoError(t, err)
		assert.Equal(t, validBucket, a.Bucket)
	})

	t.Run("rejects empty account id", func(t *testing.T) {
		_, err := NewAvatar("", validObjectKey, validBucket, validContentType, validSizeBytes)
		assert.ErrorIs(t, err, ErrEmptyAccountID)
	})

	t.Run("rejects whitespace-only account id", func(t *testing.T) {
		_, err := NewAvatar("   ", validObjectKey, validBucket, validContentType, validSizeBytes)
		assert.ErrorIs(t, err, ErrEmptyAccountID)
	})

	t.Run("rejects empty object key", func(t *testing.T) {
		_, err := NewAvatar(validAccountID, "", validBucket, validContentType, validSizeBytes)
		assert.ErrorIs(t, err, ErrEmptyObjectKey)
	})

	t.Run("rejects empty bucket", func(t *testing.T) {
		_, err := NewAvatar(validAccountID, validObjectKey, "", validContentType, validSizeBytes)
		assert.ErrorIs(t, err, ErrEmptyBucket)
	})

	t.Run("rejects unsupported content type", func(t *testing.T) {
		_, err := NewAvatar(validAccountID, validObjectKey, validBucket, "text/plain", validSizeBytes)
		assert.ErrorIs(t, err, ErrInvalidImageFormat)
	})

	t.Run("rejects zero size", func(t *testing.T) {
		_, err := NewAvatar(validAccountID, validObjectKey, validBucket, validContentType, 0)
		assert.ErrorIs(t, err, ErrInvalidImageSize)
	})

	t.Run("rejects negative size", func(t *testing.T) {
		_, err := NewAvatar(validAccountID, validObjectKey, validBucket, validContentType, -1)
		assert.ErrorIs(t, err, ErrInvalidImageSize)
	})
}

func TestIsSupportedImageType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/webp", true},
		{"image/gif", true},
		{"image/tiff", false},
		{"text/plain", false},
		{"application/json", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSupportedImageType(tt.contentType))
		})
	}
}
