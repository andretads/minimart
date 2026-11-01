//go:build integration

package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"mime/multipart"
	"minimart/internal/avatar/application"
	avatarpg "minimart/internal/avatar/infrastructure/postgres"
	avatarhttp "minimart/internal/avatar/presentation/http"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/storage/s3"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const testMaxImageSize = 5 * 1024 * 1024

type avatarHarness struct {
	svc   *application.Service
	repo  *avatarpg.AvatarRepository
	write *pgxpool.Pool
	mux   *http.ServeMux
}

func newAvatarHarness(t *testing.T) *avatarHarness {
	t.Helper()
	readPool, writePool := mustPools(t)
	client, err := s3.NewClient(s3.Config{
		Endpoint:     testS3.Endpoint,
		Bucket:       testS3.Bucket,
		Region:       "us-east-1",
		AccessKey:    testS3.AccessKey,
		SecretKey:    testS3.SecretKey,
		UsePathStyle: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		readPool.Close()
		writePool.Close()
	})

	repo := avatarpg.NewAvatarRepository(readPool, writePool)
	svc := application.NewService(repo, client, time.Hour, testMaxImageSize, metrics.Noop{})
	mux := http.NewServeMux()
	avatarhttp.NewHandlers(svc).Register(mux)
	return &avatarHarness{
		svc:   svc,
		repo:  repo,
		write: writePool,
		mux:   mux,
	}
}

func newTestAccount(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := "it-" + rand.Text()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO accounts (id, username, password_hash)
		VALUES ($1, $2, $3)
	`, id, "testuser-"+id, []byte("hash"))
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(t.Context()), `DELETE FROM accounts WHERE id = $1`, id)
	})
	return id
}

func validUploadAvatarRequest(accountID string) application.UploadAvatarRequest {
	return application.UploadAvatarRequest{
		AccountID: accountID,
		Image:     imageUpload("image/jpeg", 100),
	}
}

func imageUpload(contentType string, size int64) *application.ImageUpload {
	return &application.ImageUpload{
		Reader:      bytes.NewReader(make([]byte, int(size))),
		Filename:    "avatar.jpg",
		ContentType: contentType,
		Size:        size,
	}
}

func createTestAvatar(t *testing.T, svc *application.Service, accountID string) application.AvatarView {
	t.Helper()
	view, err := svc.UploadAvatar(t.Context(), validUploadAvatarRequest(accountID))
	require.NoError(t, err)
	return view
}

type avatarHTTPResponse struct {
	AccountID   string    `json:"account_id"`
	ImageURL    string    `json:"image_url"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func uploadAvatarHTTP(t *testing.T, h *avatarHarness, accountID, contentType string, size int64) avatarHTTPResponse {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", avatarFilename(contentType))
	require.NoError(t, err)
	_, err = fw.Write(make([]byte, int(size)))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/avatars/"+accountID, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: accountID, Subject: accountID}))

	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	return decodeAvatarHTTP(t, rec)
}

func getAvatarHTTP(t *testing.T, h *avatarHarness, accountID string) avatarHTTPResponse {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/avatars/"+accountID, nil)
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	return decodeAvatarHTTP(t, rec)
}

func decodeAvatarHTTP(t *testing.T, rec *httptest.ResponseRecorder) avatarHTTPResponse {
	t.Helper()
	var env struct {
		Data avatarHTTPResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env.Data
}

func avatarFilename(contentType string) string {
	switch contentType {
	case "image/png":
		return "avatar.png"
	case "image/webp":
		return "avatar.webp"
	case "image/gif":
		return "avatar.gif"
	default:
		return "avatar.jpg"
	}
}

type failingS3Client struct {
	uploadErr  error
	presignErr error
}

func (c *failingS3Client) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	return c.uploadErr
}

func (c *failingS3Client) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *failingS3Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if c.presignErr != nil {
		return "", c.presignErr
	}
	return "https://presigned.example.com/" + key, nil
}

func (c *failingS3Client) Bucket() string {
	return testS3.Bucket
}

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	readPool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	writePool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	return readPool, writePool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
