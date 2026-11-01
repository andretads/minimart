package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAddress(t *testing.T) {
	const testUserID = "test-user-1"
	ctx := withAuthContext(t.Context(), testUserID)
	mux, _ := newTestMux(t)

	t.Run("returns address by ID", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "+1-555-1234", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		rec = doWithContextAndHeaders(t, mux, "GET", "/v1/addresses/"+created.ID, "", ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, created.ID, data.ID)
		assert.Equal(t, "Home", data.Label)
		assert.Equal(t, "John Doe", data.FullName)
	})

	t.Run("returns 404 for non-existent address", func(t *testing.T) {
		rec := doWithContextAndHeaders(t, mux, "GET", "/v1/addresses/non-existent", "", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "endereço não encontrado", env.Error.Message)
	})
}
