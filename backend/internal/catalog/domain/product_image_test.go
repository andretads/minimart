package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProductImage(t *testing.T) {
	t.Run("builds a valid image", func(t *testing.T) {
		img, err := NewProductImage("p1", "products/p1/images/i1/a.jpg", "bucket", "image/jpeg", 10)
		require.NoError(t, err)
		assert.Equal(t, "p1", img.ProductID)
		assert.Equal(t, "products/p1/images/i1/a.jpg", img.ObjectKey)
		assert.Equal(t, "bucket", img.Bucket)
		assert.Equal(t, "image/jpeg", img.ContentType)
		assert.Equal(t, int64(10), img.SizeBytes)
		assert.Equal(t, 1, img.Version)
		assert.False(t, img.CreatedAt.IsZero())
		assert.Equal(t, img.CreatedAt, img.UpdatedAt)
	})

	tests := []struct {
		name        string
		productID   string
		objectKey   string
		bucket      string
		contentType string
		size        int64
		want        error
	}{
		{"empty product id", " ", "key", "bucket", "image/jpeg", 10, ErrEmptyProductID},
		{"empty object key", "p1", " ", "bucket", "image/jpeg", 10, ErrEmptyObjectKey},
		{"empty bucket", "p1", "key", " ", "image/jpeg", 10, ErrEmptyBucket},
		{"unsupported type", "p1", "key", "bucket", "text/plain", 10, ErrInvalidImageFormat},
		{"zero size", "p1", "key", "bucket", "image/jpeg", 0, ErrInvalidImageSize},
		{"negative size", "p1", "key", "bucket", "image/jpeg", -1, ErrInvalidImageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProductImage(tt.productID, tt.objectKey, tt.bucket, tt.contentType, tt.size)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}
