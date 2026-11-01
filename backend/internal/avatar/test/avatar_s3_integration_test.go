//go:build integration

package test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"minimart/internal/avatar/application"
	"minimart/internal/avatar/infrastructure/postgres"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/storage/s3"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	avatarhttp "minimart/internal/avatar/presentation/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestAvatarS3UploadAndPresign(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	require.NoError(t, err)
	defer pool.Close()

	accountID := "it-s3-" + uniqueAccountID()
	require.NoError(t, seedTestAccount(ctx, pool, accountID))
	defer cleanupAccount(ctx, pool, accountID)

	repo := postgres.NewAvatarRepository(pool, pool)

	// Use the real S3 client from test infrastructure
	s3Client, err := s3.NewClient(s3.Config{
		Endpoint:     testS3.Endpoint,
		Bucket:       testS3.Bucket,
		Region:       "us-east-1",
		AccessKey:    testS3.AccessKey,
		SecretKey:    testS3.SecretKey,
		UsePathStyle: true,
	})
	require.NoError(t, err)

	svc := application.NewService(repo, s3Client, time.Hour, 5*1024*1024, metrics.Noop{})

	// Upload an avatar
	view, err := svc.UploadAvatar(ctx, application.UploadAvatarRequest{
		AccountID: accountID,
		Image: &application.ImageUpload{
			Reader:      strings.NewReader("fake image data"),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        int64(len("fake image data")),
		},
	})
	require.NoError(t, err)
	require.Equal(t, accountID, view.Avatar.AccountID)
	require.NotEmpty(t, view.URL)
	require.True(t, strings.HasPrefix(view.URL, testS3.Endpoint))

	// Verify the presigned URL actually works by fetching it
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, view.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))

	// Get avatar via service
	view, err = svc.GetAvatar(ctx, application.GetAvatarRequest{AccountID: accountID})
	require.NoError(t, err)
	require.NotEmpty(t, view.URL)
	require.Equal(t, "image/jpeg", view.Avatar.ContentType)
}

func TestAvatarHTTPIntegration(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	require.NoError(t, err)
	defer pool.Close()

	accountID := "it-http-" + uniqueAccountID()
	require.NoError(t, seedTestAccount(ctx, pool, accountID))
	defer cleanupAccount(ctx, pool, accountID)

	repo := postgres.NewAvatarRepository(pool, pool)

	s3Client, err := s3.NewClient(s3.Config{
		Endpoint:     testS3.Endpoint,
		Bucket:       testS3.Bucket,
		Region:       "us-east-1",
		AccessKey:    testS3.AccessKey,
		SecretKey:    testS3.SecretKey,
		UsePathStyle: true,
	})
	require.NoError(t, err)

	svc := application.NewService(repo, s3Client, time.Hour, 5*1024*1024, metrics.Noop{})
	handlers := avatarhttp.NewHandlers(svc)

	mux := http.NewServeMux()
	handlers.Register(mux)

	// PUT upload
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("image", "test.jpg")
	_, _ = fw.Write([]byte("http test image"))
	_ = w.Close()

	req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+accountID, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	// Add auth context
	claims := &auth.Claims{UserID: accountID, Subject: accountID}
	req = req.WithContext(auth.WithClaims(ctx, claims))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Data struct {
			AccountID   string    `json:"account_id"`
			ImageURL    string    `json:"image_url,omitempty"`
			ContentType string    `json:"content_type"`
			SizeBytes   int64     `json:"size_bytes"`
			Version     int       `json:"version"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, accountID, env.Data.AccountID)
	require.NotEmpty(t, env.Data.ImageURL)
	require.Equal(t, "image/jpeg", env.Data.ContentType)

	// GET the avatar
	getReq := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/avatars/"+accountID, nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)

	var getEnv struct {
		Data struct {
			AccountID   string    `json:"account_id"`
			ImageURL    string    `json:"image_url,omitempty"`
			ContentType string    `json:"content_type"`
			SizeBytes   int64     `json:"size_bytes"`
			Version     int       `json:"version"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &getEnv))
	require.Equal(t, accountID, getEnv.Data.AccountID)
	require.NotEmpty(t, getEnv.Data.ImageURL)
}
