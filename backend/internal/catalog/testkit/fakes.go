package testkit

import (
	"context"
	"io"
	"minimart/internal/catalog/application"
	"minimart/pkg/moderation"
	"time"
)

// fakeS3Client is a test double for application.S3Client that does nothing.
type fakeS3Client struct{}

func (fakeS3Client) Upload(context.Context, string, io.Reader, string, int64) error {
	return nil
}

func (fakeS3Client) Delete(context.Context, string) error {
	return nil
}

func (fakeS3Client) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://presigned.example.com/" + key, nil
}

func (fakeS3Client) Bucket() string {
	return "test-bucket"
}

// NewFakeS3Client returns a fake S3 client for testing.
func NewFakeS3Client() application.S3Client {
	return fakeS3Client{}
}

// fakeModerationProvider is a test double for moderation.Provider that always approves.
type fakeModerationProvider struct{}

func (fakeModerationProvider) Name() string {
	return "fake"
}

func (fakeModerationProvider) Moderate(context.Context, string) (moderation.Result, error) {
	return moderation.Result{
		Decision:     "approved",
		Confidence:   1.0,
		Categories:   []string{},
		Explanation:  "Fake provider always approves",
		Model:        "fake",
		ModelVersion: "1.0",
		ModeratedAt:  time.Now().UTC(),
	}, nil
}

// NewFakeModerationProvider returns a fake moderation provider for testing.
func NewFakeModerationProvider() moderation.Provider {
	return fakeModerationProvider{}
}
