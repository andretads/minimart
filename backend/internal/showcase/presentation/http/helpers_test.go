package http

import (
	"context"
	"encoding/json"
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/internal/showcase/infrastructure/memory"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubSigner struct{}

func (stubSigner) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "https://cdn.test/" + key, nil
}

// envelope matches the standard response.Response shape for test decoding.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta *struct {
		PageSize   int    `json:"page_size"`
		NextCursor string `json:"next_cursor"`
		HasMore    *bool  `json:"has_more"`
	} `json:"meta"`
}

// newTestMux builds a test ServeMux with an in-memory repository.
func newTestMux(t *testing.T) (*http.ServeMux, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, stubSigner{}, time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux, repo
}

// do performs a request against the mux.
func do(t *testing.T, mux *http.ServeMux, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

// assertJSONResponse decodes the response and asserts status and success.
func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedSuccess bool) envelope {
	t.Helper()
	require.Equal(t, expectedStatus, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, expectedSuccess, env.Success)
	return env
}

// unmarshalData decodes the Data field into v.
func unmarshalData(t *testing.T, env envelope, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(env.Data, v))
}

// seedProduct inserts a product into the memory repository for testing.
func seedProduct(t *testing.T, repo *memory.Repository, p domain.Product) {
	t.Helper()
	require.NoError(t, repo.Save(p))
}

// seedImage inserts an image into the memory repository for testing.
func seedImage(t *testing.T, repo *memory.Repository, img domain.ProductImage) {
	t.Helper()
	require.NoError(t, repo.SaveImage(img))
}

// seedReview inserts a review into the memory repository for testing.
func seedReview(t *testing.T, repo *memory.Repository, review domain.Review) {
	t.Helper()
	require.NoError(t, repo.SaveReview(review))
}
