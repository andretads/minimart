package http

import (
	"context"
	"io"
	"minimart/internal/avatar/application"
	"minimart/internal/avatar/infrastructure/memory"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"testing"
	"time"
)

type fakeS3Client struct{}

func (fakeS3Client) Upload(context.Context, string, io.Reader, string, int64) error { return nil }
func (fakeS3Client) Delete(context.Context, string) error                           { return nil }
func (fakeS3Client) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "https://presigned.example.com/", nil
}
func (fakeS3Client) Bucket() string { return "test-bucket" }

func newTestMux(t *testing.T) (*http.ServeMux, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, fakeS3Client{}, time.Hour, 5*1024*1024, metrics.Noop{})
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux, repo
}

func withAuthContext(ctx context.Context, userID string) context.Context {
	claims := &auth.Claims{
		UserID:  userID,
		Subject: userID,
	}
	return auth.WithClaims(ctx, claims)
}
