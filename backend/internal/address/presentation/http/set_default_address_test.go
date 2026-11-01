package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultAddress(t *testing.T) {
	const testUserID = "test-user-1"
	ctx := withAuthContext(t.Context(), testUserID)
	mux, _ := newTestMux(t)

	t.Run("sets address as default", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		rec = doWithContextAndHeaders(t, mux, "POST", "/v1/addresses/"+created.ID+"/default", "", ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.True(t, data.IsDefault)
		assert.Equal(t, 2, data.Version)
	})

	t.Run("unsets old default when setting new default", func(t *testing.T) {
		// Create first address as default
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", true)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var first AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &first))

		// Create second address
		body = buildCreateAddressRequest(t, "Work", "John Doe", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", false)
		rec = doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var second AddressResponse
		env = assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &second))

		// Set second as default
		rec = doWithContextAndHeaders(t, mux, "POST", "/v1/addresses/"+second.ID+"/default", "", ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.True(t, data.IsDefault)
		assert.Equal(t, second.ID, data.ID)

		// Verify first is no longer default
		rec = doWithContextAndHeaders(t, mux, "GET", "/v1/addresses/"+first.ID, "", ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusOK, true)
		require.NotNil(t, env.Data)
		var firstData AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &firstData))
		assert.False(t, firstData.IsDefault)
	})

	t.Run("returns 404 for non-existent address", func(t *testing.T) {
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses/non-existent/default", "", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "endereço não encontrado", env.Error.Message)
	})
}
