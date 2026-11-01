package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingS3Client is a fake S3 client that records uploads and deletes.
type recordingS3Client struct {
	uploads  []uploadRecord
	deletes  []deleteRecord
	uploadFn func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
}

type uploadRecord struct {
	key         string
	body        []byte
	contentType string
	size        int64
}

type deleteRecord struct {
	key string
}

func (r *recordingS3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	if r.uploadFn != nil {
		return r.uploadFn(ctx, key, body, contentType, size)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	r.uploads = append(r.uploads, uploadRecord{
		key:         key,
		body:        data,
		contentType: contentType,
		size:        size,
	})
	return nil
}

func (r *recordingS3Client) Delete(ctx context.Context, key string) error {
	r.deletes = append(r.deletes, deleteRecord{key: key})
	return nil
}

func (r *recordingS3Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "https://presigned.example.com/" + key, nil
}

func (r *recordingS3Client) Bucket() string {
	return "test-bucket"
}

func TestValidateImage(t *testing.T) {
	svc := NewService(memory.New(), &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	tests := []struct {
		name    string
		img     *ImageUpload
		wantErr error
	}{
		{"nil image is valid", nil, nil},
		{"valid jpeg", &ImageUpload{ContentType: "image/jpeg", Size: 100}, nil},
		{"valid png", &ImageUpload{ContentType: "image/png", Size: 100}, nil},
		{"valid webp", &ImageUpload{ContentType: "image/webp", Size: 100}, nil},
		{"valid gif", &ImageUpload{ContentType: "image/gif", Size: 100}, nil},
		{"invalid content type", &ImageUpload{ContentType: "text/plain", Size: 100}, domain.ErrInvalidImageFormat},
		{"zero size", &ImageUpload{ContentType: "image/jpeg", Size: 0}, domain.ErrInvalidImageSize},
		{"negative size", &ImageUpload{ContentType: "image/jpeg", Size: -1}, domain.ErrInvalidImageSize},
		{"too large", &ImageUpload{ContentType: "image/jpeg", Size: 6 * 1024 * 1024}, domain.ErrImageTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateImage(tt.img)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStoreImage(t *testing.T) {
	svc := NewService(memory.New(), &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
	s3Client, ok := svc.s3Client.(*recordingS3Client)
	require.True(t, ok)

	t.Run("uploads image and returns key", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("fake image data")),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        100,
		}

		key, err := svc.storeImage(t.Context(), "prod-123", "img-1", img)
		require.NoError(t, err)

		assert.Equal(t, "products/prod-123/images/img-1/test.jpg", key)
		require.Len(t, s3Client.uploads, 1)
		assert.Equal(t, "products/prod-123/images/img-1/test.jpg", s3Client.uploads[0].key)
		assert.Equal(t, "image/jpeg", s3Client.uploads[0].contentType)
		assert.Equal(t, int64(100), s3Client.uploads[0].size)
		assert.Equal(t, []byte("fake image data"), s3Client.uploads[0].body)
	})

	t.Run("sanitizes filename", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("data")),
			Filename:    "my image@!.jpg",
			ContentType: "image/jpeg",
			Size:        10,
		}

		key, err := svc.storeImage(t.Context(), "prod-456", "img-1", img)
		require.NoError(t, err)
		assert.Equal(t, "products/prod-456/images/img-1/my_image__.jpg", key)
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("data")),
			Filename:    "test.txt",
			ContentType: "text/plain",
			Size:        10,
		}

		_, err := svc.storeImage(t.Context(), "prod-789", "img-1", img)
		assert.ErrorIs(t, err, domain.ErrInvalidImageFormat)
	})

	t.Run("rejects too large image", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("data")),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        6 * 1024 * 1024,
		}

		_, err := svc.storeImage(t.Context(), "prod-999", "img-1", img)
		assert.ErrorIs(t, err, domain.ErrImageTooLarge)
	})

	t.Run("wraps upload errors", func(t *testing.T) {
		boom := errors.New("network down")
		failClient := &recordingS3Client{}
		failClient.uploadFn = func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			return boom
		}
		svc := NewService(memory.New(), failClient, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("data")),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        10,
		}

		_, err := svc.storeImage(t.Context(), "prod-111", "img-1", img)
		require.ErrorIs(t, err, boom)
		assert.Contains(t, err.Error(), "upload image")
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple.jpg", "simple.jpg"},
		{"my file.jpg", "my_file.jpg"},
		{"file@#$.png", "file___.png"},
		{"normal-file_name.webp", "normal-file_name.webp"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeFilename(tt.input))
		})
	}
}
