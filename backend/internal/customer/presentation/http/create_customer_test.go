package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCustomerStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "corpo JSON inválido", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing user_id", body: `{"display_name":"John"}`, wantStatus: http.StatusBadRequest},
		{name: "display_name too long", body: `{"user_id":"user-1","display_name":"` + string(make([]byte, 201)) + `"}`, wantStatus: http.StatusBadRequest},
		{name: "valid", body: `{"user_id":"user-1","display_name":"John"}`, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, _ := newTestMux(t)
			rec := do(t, mux, http.MethodPost, "/v1/customers", tt.body)
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

func TestCreateCustomerReturnsBody(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := do(t, mux, http.MethodPost, "/v1/customers", `{"user_id":"user-1","display_name":"John"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, `"1"`, rec.Header().Get("ETag"), "a new customer exposes its version as an ETag")

	var env struct {
		Data customerResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.NotEmpty(t, env.Data.ID)
	assert.Equal(t, "user-1", env.Data.UserID)
	assert.Equal(t, "John", env.Data.DisplayName)
	assert.Equal(t, 1, env.Data.Version, "a new customer starts at version 1")
	assert.NotEmpty(t, env.Data.CreatedAt, "created_at is returned")
	assert.NotEmpty(t, env.Data.UpdatedAt, "updated_at is returned")
}
