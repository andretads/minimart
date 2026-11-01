package http

import (
	"encoding/json"
	"minimart/internal/customer/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateCustomerStatusMapping(t *testing.T) {
	mux, repo := newTestMux(t)
	c := domain.Customer{
		ID:          "c1",
		UserID:      "user-1",
		DisplayName: "John",
		Status:      domain.StatusActive,
		Version:     1,
	}
	require.NoError(t, repo.Save(t.Context(), c))

	t.Run("deactivates an active customer", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/customers/c1/deactivate", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, `"2"`, rec.Header().Get("ETag"), "deactivate bumps the version and exposes it as an ETag")

		var env struct {
			Data customerResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "c1", env.Data.ID)
		assert.Equal(t, "deactivated", env.Data.Status)
		assert.NotEmpty(t, env.Data.DeactivatedAt)
		assert.Equal(t, 2, env.Data.Version, "deactivate bumps the version")
	})

	t.Run("idempotent: deactivating already deactivated customer returns it unchanged (status/deactivated_at), version bumps", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/customers/c1/deactivate", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data customerResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Equal(t, "c1", env.Data.ID)
		assert.Equal(t, "deactivated", env.Data.Status)
		assert.NotEmpty(t, env.Data.DeactivatedAt)
		assert.Equal(t, 3, env.Data.Version, "second deactivate bumps version again")
	})

	t.Run("returns 404 for unknown ID", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/customers/missing/deactivate", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
