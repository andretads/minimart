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

func addImage(t *testing.T, mux *http.ServeMux, productID, filename string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", filename)
	require.NoError(t, err)
	_, err = fw.Write(body)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/products/"+productID+"/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAddProductImage(t *testing.T) {
	t.Run("appends images and bumps the product version", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		rec := addImage(t, mux, "p1", "first.jpg", []byte("first"))
		require.Equal(t, http.StatusCreated, rec.Code)

		var first struct {
			Data productImageResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &first))
		assert.Equal(t, "p1", first.Data.ProductID)
		assert.Equal(t, 0, first.Data.Position)
		assert.Equal(t, "image/jpeg", first.Data.ContentType)
		assert.NotEmpty(t, first.Data.URL)

		rec = addImage(t, mux, "p1", "second.png", []byte("second"))
		require.Equal(t, http.StatusCreated, rec.Code)
		var second struct {
			Data productImageResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &second))
		assert.Equal(t, 1, second.Data.Position)

		stored, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, stored.Version, "each image write bumps the product version")
	})

	t.Run("returns 404 for a missing product", func(t *testing.T) {
		mux, _ := newTestMux(t)
		rec := addImage(t, mux, "missing", "first.jpg", []byte("first"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("rejects a missing image", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		require.NoError(t, w.Close())
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/products/p1/images", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects an unsupported image type", func(t *testing.T) {
		mux, repo := newTestMux(t)
		p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), p))

		rec := addImage(t, mux, "p1", "notes.txt", []byte("not an image"))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestListProductImages(t *testing.T) {
	mux, repo := newTestMux(t)
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	require.Equal(t, http.StatusCreated, addImage(t, mux, "p1", "first.jpg", []byte("first")).Code)
	require.Equal(t, http.StatusCreated, addImage(t, mux, "p1", "second.png", []byte("second")).Code)

	rec := do(t, mux, http.MethodGet, "/v1/products/p1/images", "")
	require.Equal(t, http.StatusOK, rec.Code)

	var env struct {
		Data []productImageResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Len(t, env.Data, 2)
	assert.Equal(t, 0, env.Data[0].Position)
	assert.Equal(t, 1, env.Data[1].Position)

	productRec := do(t, mux, http.MethodGet, "/v1/products/p1", "")
	require.Equal(t, http.StatusOK, productRec.Code)
	var productEnv struct {
		Data productResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(productRec.Body.Bytes(), &productEnv))
	require.Len(t, productEnv.Data.Images, 2)
	assert.Equal(t, 0, productEnv.Data.Images[0].Position)

	t.Run("returns 404 for a missing product", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/products/missing/images", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestDeleteProductImage(t *testing.T) {
	mux, repo := newTestMux(t)
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))

	rec := addImage(t, mux, "p1", "first.jpg", []byte("first"))
	require.Equal(t, http.StatusCreated, rec.Code)
	var added struct {
		Data productImageResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &added))

	rec = do(t, mux, http.MethodDelete, "/v1/products/p1/images/"+added.Data.ID, "")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	stored, err := repo.GetByID(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, 3, stored.Version, "delete bumps the product version")

	rec = do(t, mux, http.MethodGet, "/v1/products/p1/images", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Data []productImageResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Empty(t, env.Data)

	t.Run("returns 404 for an unknown image", func(t *testing.T) {
		rec := do(t, mux, http.MethodDelete, "/v1/products/p1/images/missing", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
