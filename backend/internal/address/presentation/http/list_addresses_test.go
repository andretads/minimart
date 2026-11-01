package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAddresses(t *testing.T) {
	const testUserID = "test-user-1"
	ctx := withAuthContext(t.Context(), testUserID)
	mux, _ := newTestMux(t)

	t.Run("returns empty list for customer with no addresses", func(t *testing.T) {
		rec := doWithContextAndHeaders(t, mux, "GET", "/v1/addresses", "", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data []AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Empty(t, data)
	})

	t.Run("returns all addresses ordered by is_default DESC, created_at DESC", func(t *testing.T) {
		// Create non-default first
		body := buildCreateAddressRequest(t, "Work", "John Doe", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		// Create default
		body = buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", true)
		rec = doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		// Create another non-default
		body = buildCreateAddressRequest(t, "Other", "John Doe", "789 Other St", "Chicago", "IL", "60601", "USA", "", false)
		rec = doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		rec = doWithContextAndHeaders(t, mux, "GET", "/v1/addresses", "", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data []AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		require.Len(t, data, 3)
		// Default should be first
		assert.Equal(t, "Home", data[0].Label)
		assert.True(t, data[0].IsDefault)
		// Then the rest in created_at DESC order
		assert.Equal(t, "Other", data[1].Label)
		assert.Equal(t, "Work", data[2].Label)
	})
}
