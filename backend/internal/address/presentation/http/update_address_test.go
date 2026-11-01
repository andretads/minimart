package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAddress(t *testing.T) {
	const testUserID = "test-user-1"
	ctx := withAuthContext(t.Context(), testUserID)
	mux, _ := newTestMux(t)

	t.Run("updates an address with valid version", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "+1-555-1234", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))
		fmt.Println("Created ID:", created.ID, "Version:", created.Version)

		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "+1-555-5678", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		fmt.Println("PATCH Response:", rec.Code, rec.Body.String())
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Work", data.Label)
		assert.Equal(t, "Jane Smith", data.FullName)
		assert.Equal(t, "456 Office Ave", data.Street)
		assert.Equal(t, "San Francisco", data.City)
		assert.Equal(t, "CA", data.State)
		assert.Equal(t, "94102", data.Zip)
		assert.Equal(t, "USA", data.Country)
		assert.Equal(t, "+1-555-5678", data.Phone)
		assert.Equal(t, 2, data.Version)

		etag := rec.Header().Get("ETag")
		assert.Equal(t, `"2"`, etag)
	})

	t.Run("returns 409 for stale version", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "+1-555-1234", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		// First update
		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "+1-555-5678", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		require.Equal(t, http.StatusOK, rec.Code)

		// Second update with stale version
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusConflict, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "endereço foi modificado por outra requisição; recarregue e tente novamente", env.Error.Message)
	})

	t.Run("uses If-Match header when present", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "+1-555-1234", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		// Update with If-Match header
		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "+1-555-5678", 999) // wrong version in body
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, map[string]string{
			"If-Match": `"1"`, // correct version in header
		})
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Work", data.Label)
		assert.Equal(t, 2, data.Version)
	})

	t.Run("rejects invalid If-Match header", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, map[string]string{
			"If-Match": `invalid`,
		})
		env = assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match inválido", env.Error.Message)
	})

	t.Run("rejects wildcard If-Match header", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, map[string]string{
			"If-Match": `*`,
		})
		env = assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cabeçalho If-Match inválido", env.Error.Message)
	})

	t.Run("returns 404 for non-existent address", func(t *testing.T) {
		updateBody := buildUpdateAddressRequest(t, "Work", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", 1)
		rec := doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/non-existent", updateBody, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusNotFound, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "endereço não encontrado", env.Error.Message)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		rec := doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/addr-1", "{invalid", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "corpo JSON inválido", env.Error.Message)
	})

	t.Run("rejects empty label", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		updateBody := buildUpdateAddressRequest(t, "   ", "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "rótulo é obrigatório", env.Error.Message)
	})

	t.Run("rejects label too long", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		longLabel := strings.Repeat("a", 51)
		updateBody := buildUpdateAddressRequest(t, longLabel, "Jane Smith", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "rótulo excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		updateBody := buildUpdateAddressRequest(t, "Work", "   ", "456 Office Ave", "San Francisco", "CA", "94102", "USA", "", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "nome completo é obrigatório", env.Error.Message)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		require.Equal(t, http.StatusCreated, rec.Code)

		var created AddressResponse
		env := assertJSONResponse(t, rec, http.StatusCreated, true)
		require.NoError(t, json.Unmarshal(env.Data, &created))

		updateBody := buildUpdateAddressRequest(t, "  Work  ", "  Jane Smith  ", "  456 Office Ave  ", "  San Francisco  ", "  CA  ", "  94102  ", "  USA  ", "  +1-555-5678  ", created.Version)
		rec = doWithContextAndHeaders(t, mux, "PATCH", "/v1/addresses/"+created.ID, updateBody, ctx, nil)
		env = assertJSONResponse(t, rec, http.StatusOK, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Work", data.Label)
		assert.Equal(t, "Jane Smith", data.FullName)
		assert.Equal(t, "456 Office Ave", data.Street)
		assert.Equal(t, "San Francisco", data.City)
		assert.Equal(t, "CA", data.State)
		assert.Equal(t, "94102", data.Zip)
		assert.Equal(t, "USA", data.Country)
		assert.Equal(t, "+1-555-5678", data.Phone)
	})
}
