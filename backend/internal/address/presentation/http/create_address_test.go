package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAddress(t *testing.T) {
	const testUserID = "test-user-1"
	ctx := withAuthContext(t.Context(), testUserID)
	mux, _ := newTestMux(t)

	t.Run("creates an address with all fields", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "+1-555-1234", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.NotEmpty(t, data.ID)
		assert.Equal(t, "Home", data.Label)
		assert.Equal(t, "John Doe", data.FullName)
		assert.Equal(t, "123 Main St", data.Street)
		assert.Equal(t, "New York", data.City)
		assert.Equal(t, "NY", data.State)
		assert.Equal(t, "10001", data.Zip)
		assert.Equal(t, "USA", data.Country)
		assert.Equal(t, "+1-555-1234", data.Phone)
		assert.False(t, data.IsDefault)
		assert.Equal(t, 1, data.Version)

		etag := rec.Header().Get("ETag")
		assert.Equal(t, `"1"`, etag)
	})

	t.Run("creates an address without phone", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Home", data.Label)
		assert.Empty(t, data.Phone)
	})

	t.Run("creates an address as default", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", true)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.True(t, data.IsDefault)
	})

	t.Run("rejects empty label", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "   ", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "rótulo é obrigatório", env.Error.Message)
	})

	t.Run("rejects label too long", func(t *testing.T) {
		longLabel := strings.Repeat("a", 51)
		body := buildCreateAddressRequest(t, longLabel, "John Doe", "123 Main St", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "rótulo excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty full name", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "   ", "123 Main St", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "nome completo é obrigatório", env.Error.Message)
	})

	t.Run("rejects full name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 101)
		body := buildCreateAddressRequest(t, "Home", longName, "123 Main St", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "nome completo excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty street", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "   ", "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "logradouro é obrigatório", env.Error.Message)
	})

	t.Run("rejects street too long", func(t *testing.T) {
		longStreet := strings.Repeat("a", 201)
		body := buildCreateAddressRequest(t, "Home", "John Doe", longStreet, "New York", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "logradouro excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty city", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "   ", "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cidade é obrigatória", env.Error.Message)
	})

	t.Run("rejects city too long", func(t *testing.T) {
		longCity := strings.Repeat("a", 101)
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", longCity, "NY", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "cidade excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty state", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "   ", "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "estado é obrigatório", env.Error.Message)
	})

	t.Run("rejects state too long", func(t *testing.T) {
		longState := strings.Repeat("a", 101)
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", longState, "10001", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "estado excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty zip", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "   ", "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "CEP é obrigatório", env.Error.Message)
	})

	t.Run("rejects zip too long", func(t *testing.T) {
		longZip := strings.Repeat("a", 21)
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", longZip, "USA", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "CEP excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects empty country", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "   ", "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "país é obrigatório", env.Error.Message)
	})

	t.Run("rejects country too long", func(t *testing.T) {
		longCountry := strings.Repeat("a", 101)
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", longCountry, "", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "país excede o tamanho máximo", env.Error.Message)
	})

	t.Run("rejects phone too long", func(t *testing.T) {
		longPhone := strings.Repeat("a", 31)
		body := buildCreateAddressRequest(t, "Home", "John Doe", "123 Main St", "New York", "NY", "10001", "USA", longPhone, false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "telefone excede o tamanho máximo", env.Error.Message)
	})

	t.Run("trims whitespace from all fields", func(t *testing.T) {
		body := buildCreateAddressRequest(t, "  Home  ", "  John Doe  ", "  123 Main St  ", "  New York  ", "  NY  ", "  10001  ", "  USA  ", "  +1-555-1234  ", false)

		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", body, ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusCreated, true)

		require.NotNil(t, env.Data)
		var data AddressResponse
		require.NoError(t, json.Unmarshal(env.Data, &data))
		assert.Equal(t, "Home", data.Label)
		assert.Equal(t, "John Doe", data.FullName)
		assert.Equal(t, "123 Main St", data.Street)
		assert.Equal(t, "New York", data.City)
		assert.Equal(t, "NY", data.State)
		assert.Equal(t, "10001", data.Zip)
		assert.Equal(t, "USA", data.Country)
		assert.Equal(t, "+1-555-1234", data.Phone)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		rec := doWithContextAndHeaders(t, mux, "POST", "/v1/addresses", "{invalid", ctx, nil)
		env := assertJSONResponse(t, rec, http.StatusBadRequest, false)
		require.NotNil(t, env.Error)
		assert.Equal(t, "corpo JSON inválido", env.Error.Message)
	})
}
