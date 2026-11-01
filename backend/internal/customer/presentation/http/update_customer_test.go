package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateCustomer(t *testing.T) {
	t.Run("updates customer display name", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCustomerRequest(t, "user-1", "John")
		rec := do(t, mux, "POST", "/v1/customers", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		var created struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created))

		rec = do(t, mux, "PATCH", "/v1/customers/"+created.ID, buildUpdateCustomerRequest(t, "Jane", created.Version))
		assertJSONResponse(t, rec, http.StatusOK, true)
	})

	t.Run("returns 404 for unknown ID", func(t *testing.T) {
		mux, _ := newTestMux(t)
		rec := do(t, mux, "PATCH", "/v1/customers/unknown", buildUpdateCustomerRequest(t, "Jane", 1))
		assertJSONResponse(t, rec, http.StatusNotFound, false)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		mux, _ := newTestMux(t)
		rec := do(t, mux, "PATCH", "/v1/customers/unknown", "{invalid}")
		assertJSONResponse(t, rec, http.StatusBadRequest, false)
	})

	t.Run("rejects missing version", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCustomerRequest(t, "user-1", "John")
		rec := do(t, mux, "POST", "/v1/customers", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		var created struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created))

		rec = do(t, mux, "PATCH", "/v1/customers/"+created.ID, buildUpdateCustomerRequest(t, "Jane", 0))
		assertJSONResponse(t, rec, http.StatusBadRequest, false)
	})

	t.Run("rejects stale version", func(t *testing.T) {
		mux, _ := newTestMux(t)
		body := buildCreateCustomerRequest(t, "user-1", "John")
		rec := do(t, mux, "POST", "/v1/customers", body)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		var created struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created))

		// First update succeeds
		rec = do(t, mux, "PATCH", "/v1/customers/"+created.ID, buildUpdateCustomerRequest(t, "Jane", created.Version))
		assertJSONResponse(t, rec, http.StatusOK, true)

		// Second update with stale version fails
		rec = do(t, mux, "PATCH", "/v1/customers/"+created.ID, buildUpdateCustomerRequest(t, "Jane2", created.Version))
		assertJSONResponse(t, rec, http.StatusConflict, false)
	})
}
