package http

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"minimart/internal/catalog/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateProductStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
	}{
		{name: "corpo JSON inválido", id: "p1", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "empty name", id: "p1", body: `{"name":"","price_cents":100,"version":1,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "non-positive price", id: "p1", body: `{"name":"Gizmo","price_cents":0,"version":1,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "missing version", id: "p1", body: `{"name":"Gizmo","price_cents":100,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "non-positive version", id: "p1", body: `{"name":"Gizmo","price_cents":100,"version":0,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "stale version", id: "p1", body: `{"name":"Gizmo","price_cents":100,"version":99,"category_id":"cat-1"}`, wantStatus: http.StatusConflict},
		{name: "missing returns 404", id: "missing", body: `{"name":"Gizmo","price_cents":100,"version":1,"category_id":"cat-1"}`, wantStatus: http.StatusNotFound},
		{name: "valid", id: "p1", body: `{"name":"Gizmo","price_cents":100,"version":1,"category_id":"cat-1"}`, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, repo := newTestMux(t)
			p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
			require.NoError(t, err)
			require.NoError(t, repo.Save(t.Context(), p))

			rec := do(t, mux, http.MethodPatch, "/v1/products/"+tt.id, tt.body)
			assert.Equal(t, tt.wantStatus, rec.Code)

			var env envelope
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			if tt.wantStatus == http.StatusOK {
				assert.True(t, env.Success)
			} else {
				assert.False(t, env.Success)
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantStatus, env.Error.Code)
			}
		})
	}
}

func TestUpdateProductReturnsBodyAndPreservesStock(t *testing.T) {
	mux, repo := newTestMux(t)
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	rec := do(t, mux, http.MethodPatch, "/v1/products/p1", `{"name":"Gizmo","price_cents":100,"version":1,"category_id":"cat-1"}`)
	require.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Data productResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "p1", env.Data.ID)
	assert.Equal(t, "Gizmo", env.Data.Name)
	assert.Equal(t, int64(100), env.Data.PriceCents)
	assert.Equal(t, 3, env.Data.Stock, "stock must be preserved")
	assert.Equal(t, 2, env.Data.Version, "an update bumps the version")
	assert.True(t, env.Data.CreatedAt.Equal(p.CreatedAt), "created_at must be preserved")
	assert.False(t, env.Data.UpdatedAt.Before(p.UpdatedAt), "updated_at must not move backwards")
	assert.Equal(t, `"2"`, rec.Header().Get("ETag"), "the response exposes the new version")
}

func TestUpdateProductIfMatch(t *testing.T) {
	tests := []struct {
		name       string
		ifMatch    string
		body       string
		wantStatus int
	}{
		{name: "matching tag overrides the body version", ifMatch: `"1"`, body: `{"name":"Gizmo","price_cents":100,"category_id":"cat-1"}`, wantStatus: http.StatusOK},
		{name: "matching tag wins over a conflicting body version", ifMatch: `"1"`, body: `{"name":"Gizmo","price_cents":100,"version":99,"category_id":"cat-1"}`, wantStatus: http.StatusOK},
		{name: "stale tag conflicts", ifMatch: `"99"`, body: `{"name":"Gizmo","price_cents":100,"version":1,"category_id":"cat-1"}`, wantStatus: http.StatusConflict},
		{name: "malformed tag is rejected", ifMatch: `not-a-tag`, body: `{"name":"Gizmo","price_cents":100,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
		{name: "wildcard is rejected", ifMatch: `*`, body: `{"name":"Gizmo","price_cents":100,"category_id":"cat-1"}`, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, repo := newTestMux(t)
			p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
			require.NoError(t, err)
			require.NoError(t, repo.Save(t.Context(), p))

			rec := doWithHeaders(t, mux, http.MethodPatch, "/v1/products/p1", tt.body, map[string]string{"If-Match": tt.ifMatch})
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, `"2"`, rec.Header().Get("ETag"))
			}
		})
	}
}

func TestUpdateProductMultipart(t *testing.T) {
	t.Run("updates product from multipart fields", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("name", "Gizmo Mk II")
		_ = w.WriteField("price_cents", "150")
		_ = w.WriteField("version", "1")
		_ = w.WriteField("category_id", "cat-1")
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/v1/products/p1", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data productResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "Gizmo Mk II", env.Data.Name)
		assert.Equal(t, int64(150), env.Data.PriceCents)
		assert.Equal(t, 2, env.Data.Version)
		assert.Empty(t, env.Data.Images)
	})

	t.Run("rejects invalid version field", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("name", "Gizmo")
		_ = w.WriteField("price_cents", "100")
		_ = w.WriteField("version", "not-a-number")
		_ = w.WriteField("category_id", "cat-1")
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/v1/products/p1", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var env envelope
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Success)
		assert.Contains(t, env.Error.Message, "version must be an integer")
	})

	t.Run("If-Match header overrides body version", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		_ = w.WriteField("name", "Gizmo")
		_ = w.WriteField("price_cents", "100")
		_ = w.WriteField("version", "99")
		_ = w.WriteField("category_id", "cat-1")
		_ = w.Close()

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/v1/products/p1", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("If-Match", `"1"`)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "If-Match should override stale body version")
	})
}
