package application

import (
	"context"
	"io"
	"minimart/internal/avatar/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"
)

// recordingS3Client is a fake S3 client that records uploads and presigns.
type recordingS3Client struct {
	uploads   []uploadRecord
	deletes   []deleteRecord
	presigns  []presignRecord
	uploadFn  func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	deleteFn  func(ctx context.Context, key string) error
	presignFn func(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type deleteRecord struct {
	key string
}

type uploadRecord struct {
	key         string
	body        []byte
	contentType string
	size        int64
}

type presignRecord struct {
	key    string
	expiry time.Duration
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
	if r.deleteFn != nil {
		return r.deleteFn(ctx, key)
	}
	r.deletes = append(r.deletes, deleteRecord{key: key})
	return nil
}

func (r *recordingS3Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	r.presigns = append(r.presigns, presignRecord{key: key, expiry: expiry})
	if r.presignFn != nil {
		return r.presignFn(ctx, key, expiry)
	}
	return "https://presigned.example.com/" + key, nil
}

func (r *recordingS3Client) Bucket() string {
	return "test-bucket"
}

// newTestService creates a service with a recording S3 client for testing.
func newTestService(t *testing.T, maxImageSize int64) (*Service, *recordingS3Client, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	s3Client := &recordingS3Client{}
	svc := NewService(repo, s3Client, time.Hour, maxImageSize, metrics.Noop{})
	return svc, s3Client, repo
}
