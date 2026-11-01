package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAvatar(t *testing.T) {
	const testAccountID = "test-account-1"

	t.Run("uploads avatar successfully", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("image", "test.jpg")
		fw, _ := w.CreateFormFile("image", "test.jpg")
		_, _ = fw.Write([]byte("fake image data"))
		_ = w.Close()

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data avatarResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, testAccountID, env.Data.AccountID)
		assert.NotEmpty(t, env.Data.ImageURL)
		assert.Equal(t, "image/jpeg", env.Data.ContentType)
		assert.Equal(t, int64(15), env.Data.SizeBytes)
		assert.Equal(t, 1, env.Data.Version)
		assert.Equal(t, `"1"`, rec.Header().Get("ETag"))
	})

	t.Run("rejects non-multipart request", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, strings.NewReader(`{"foo":"bar"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects missing image", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.Close()

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("image", "test.txt")
		_, _ = fw.Write([]byte("not an image"))
		_ = w.Close()

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects 401 without auth", func(t *testing.T) {
		mux, _ := newTestMux(t)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("image", "test.jpg")
		_, _ = fw.Write([]byte("fake image data"))
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects 403 when modifying another account's avatar", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), "other-account")

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("image", "test.jpg")
		_, _ = fw.Write([]byte("fake image data"))
		_ = w.Close()

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("rejects image too large", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("image", "test.jpg")
		// 6MB data
		_, _ = fw.Write(bytes.Repeat([]byte("x"), 6*1024*1024))
		_ = w.Close()

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestGetAvatar(t *testing.T) {
	const testAccountID = "test-account-1"

	t.Run("returns avatar with presigned URL", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		// First upload an avatar
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("image", "test.jpg")
		_, _ = fw.Write([]byte("fake image data"))
		_ = w.Close()

		uploadReq := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		uploadReq.Header.Set("Content-Type", w.FormDataContentType())
		uploadRec := httptest.NewRecorder()
		mux.ServeHTTP(uploadRec, uploadReq)
		require.Equal(t, http.StatusOK, uploadRec.Code)

		// Now get the avatar
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/avatars/"+testAccountID, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data avatarResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, testAccountID, env.Data.AccountID)
		assert.NotEmpty(t, env.Data.ImageURL)
		assert.Equal(t, "image/jpeg", env.Data.ContentType)
		assert.Equal(t, 1, env.Data.Version)
		assert.Equal(t, `"1"`, rec.Header().Get("ETag"))
	})

	t.Run("returns 404 when avatar not found", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/avatars/nonexistent", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
