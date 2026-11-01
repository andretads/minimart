package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvatar(t *testing.T) {
	const testAccountID = "test-account-1"

	t.Run("returns avatar with presigned URL", func(t *testing.T) {
		mux, _ := newTestMux(t)
		ctx := withAuthContext(t.Context(), testAccountID)

		uploadRec := uploadAvatar(t, mux, ctx, testAccountID, "test.jpg", []byte("fake image data"))
		require.Equal(t, http.StatusOK, uploadRec.Code)

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
