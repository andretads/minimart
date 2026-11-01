package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProductStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "corpo JSON inválido", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "empty name", body: `{"name":"","price_cents":100,"stock":1,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "non-positive price", body: `{"name":"Gizmo","price_cents":0,"stock":1,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "negative stock", body: `{"name":"Gizmo","price_cents":100,"stock":-1,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "missing category", body: `{"name":"Gizmo","price_cents":100,"stock":1}`, wantStatus: http.StatusBadRequest},
		{name: "valid", body: `{"name":"Gizmo","price_cents":100,"stock":1,"category_id":"cat-1"}`, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, _ := newTestMux(t)
			rec := do(t, mux, http.MethodPost, "/v1/products", tt.body)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			if tt.wantStatus == http.StatusCreated {
				assert.True(t, env.Success)
			} else {
				assert.False(t, env.Success)
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantStatus, env.Error.Code)
			}
		})
	}
}

func TestCreateProductReturnsBody(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/products", `{"name":"Gizmo","price_cents":100,"stock":1,"category_id":"cat-1"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, `"1"`, rec.Header().Get("ETag"), "a new product exposes its version as an ETag")

	var env struct {
		Data productResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Data.ID)
	assert.Equal(t, "Gizmo", env.Data.Name)
	assert.Equal(t, int64(100), env.Data.PriceCents)
	assert.Equal(t, 1, env.Data.Stock)
	assert.Equal(t, 1, env.Data.Version, "a new product starts at version 1")
	assert.False(t, env.Data.CreatedAt.IsZero(), "created_at is returned")
	assert.False(t, env.Data.UpdatedAt.IsZero(), "updated_at is returned")
}

func TestCreateProductMultipart(t *testing.T) {
	t.Run("creates product from multipart fields", func(t *testing.T) {
		mux, _ := newTestMux(t)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("name", "Gizmo")
		_ = w.WriteField("price_cents", "100")
		_ = w.WriteField("stock", "1")
		_ = w.WriteField("category_id", "cat-1")
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/products", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var env struct {
			Data productResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "Gizmo", env.Data.Name)
		assert.Equal(t, int64(100), env.Data.PriceCents)
		assert.Equal(t, 1, env.Data.Stock)
		assert.Equal(t, 1, env.Data.Version)
		assert.Empty(t, env.Data.Images, "product creation does not accept images")
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		mux, _ := newTestMux(t)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("name", "Gizmo")
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/products", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var env envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Success)
	})
}
