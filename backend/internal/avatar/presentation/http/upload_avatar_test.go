package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uploadAvatar(t *testing.T, mux *http.ServeMux, ctx context.Context, accountID, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", filename)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+accountID, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestUploadAvatar(t *testing.T) {
	const testAccountID = "test-account-1"

	t.Run("uploads avatar successfully", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		rec := uploadAvatar(t, mux, ctx, testAccountID, "test.jpg", []byte("fake image data"))

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
		require.NoError(t, w.Close())

		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/v1/avatars/"+testAccountID, &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		rec := uploadAvatar(t, mux, ctx, testAccountID, "test.txt", []byte("not an image"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects 401 without auth", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := uploadAvatar(t, mux, t.Context(), testAccountID, "test.jpg", []byte("fake image data"))

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects 403 when modifying another account's avatar", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), "other-account")

		rec := uploadAvatar(t, mux, ctx, testAccountID, "test.jpg", []byte("fake image data"))

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("rejects image too large", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		rec := uploadAvatar(t, mux, ctx, testAccountID, "test.jpg", bytes.Repeat([]byte("x"), 6*1024*1024))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
