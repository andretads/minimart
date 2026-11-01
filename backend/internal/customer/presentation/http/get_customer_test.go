package http

import (
	"encoding/json"
	"minimart/internal/customer/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCustomerStatusMapping(t *testing.T) {
	mux, repo := newTestMux(t)
	c := domain.Customer{
		ID:          "c1",
		UserID:      "user-1",
		DisplayName: "John",
		Status:      domain.StatusActive,
		Version:     1,
	}
	require.NoError(t, repo.Save(t.Context(), c))

	t.Run("found returns the customer", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/customers/c1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, `"1"`, rec.Header().Get("ETag"), "GET exposes the current version as an ETag")

		var env struct {
			Data customerResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "c1", env.Data.ID)
		assert.Equal(t, "user-1", env.Data.UserID)
		assert.Equal(t, "John", env.Data.DisplayName)
		assert.Equal(t, 1, env.Data.Version)
	})

	t.Run("missing returns 404", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/customers/missing", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestGetCustomerConditional(t *testing.T) {
	mux, repo := newTestMux(t)
	c := domain.Customer{
		ID:          "c1",
		UserID:      "user-1",
		DisplayName: "John",
		Status:      domain.StatusActive,
		Version:     1,
	}
	require.NoError(t, repo.Save(t.Context(), c))

	tests := []struct {
		name       string
		ifNone     string
		wantStatus int
	}{
		{name: "no header returns the body", wantStatus: http.StatusOK},
		{name: "matching strong tag is not modified", ifNone: `"1"`, wantStatus: http.StatusNotModified},
		{name: "matching weak tag is not modified", ifNone: `W/"1"`, wantStatus: http.StatusNotModified},
		{name: "wildcard matches an existing resource", ifNone: `*`, wantStatus: http.StatusNotModified},
		{name: "stale tag returns the body", ifNone: `"2"`, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.ifNone != "" {
				headers["If-None-Match"] = tt.ifNone
			}
			rec := doWithHeaders(t, mux, http.MethodGet, "/v1/customers/c1", "", headers)
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNotModified {
				assert.Empty(t, rec.Body.String(), "304 carries no body")
			}
		})
	}
}
