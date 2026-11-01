package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMethods(t *testing.T) {
	env := newTestEnv(t)

	t.Run("empty returns an empty list", func(t *testing.T) {
		rec := env.do(t, http.MethodGet, "/v1/shipping/methods", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var methods []ShippingMethodResponse
		require.NoError(t, json.Unmarshal(envl.Data, &methods))
		assert.Empty(t, methods)
	})

	t.Run("returns only active methods", func(t *testing.T) {
		active := env.seedMethod(t, true)
		env.seedMethod(t, false)

		rec := env.do(t, http.MethodGet, "/v1/shipping/methods", "")
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var methods []ShippingMethodResponse
		require.NoError(t, json.Unmarshal(envl.Data, &methods))
		require.Len(t, methods, 1)
		assert.Equal(t, active.ID, methods[0].ID)
		assert.True(t, methods[0].Active)
	})
}

func TestCreateMethod(t *testing.T) {
	t.Run("valid request returns 201", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/methods",
			`{"name":"Express","description":"Fast delivery","active":true,"estimated_min_days":1,"estimated_max_days":2}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var method ShippingMethodResponse
		require.NoError(t, json.Unmarshal(envl.Data, &method))
		assert.NotEmpty(t, method.ID)
		assert.Equal(t, "Express", method.Name)
		assert.Equal(t, 1, method.EstimatedMinDays)
		assert.Equal(t, 2, method.EstimatedMaxDays)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/methods", `{"name":`)
		require.Equal(t, http.StatusBadRequest, rec.Code)

		envl := decodeEnvelope(t, rec)
		assert.False(t, envl.Success)
		require.NotNil(t, envl.Error)
		assert.Equal(t, http.StatusBadRequest, envl.Error.Code)
	})

	t.Run("invalid day range returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/methods",
			`{"name":"Bad","active":true,"estimated_min_days":5,"estimated_max_days":2}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUpdateMethod(t *testing.T) {
	t.Run("valid request updates the method", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)

		rec := env.do(t, http.MethodPatch, "/v1/shipping/methods/"+method.ID,
			`{"name":"Economy","description":"Slow","active":false,"estimated_min_days":7,"estimated_max_days":10}`)
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var updated ShippingMethodResponse
		require.NoError(t, json.Unmarshal(envl.Data, &updated))
		assert.Equal(t, method.ID, updated.ID)
		assert.Equal(t, "Economy", updated.Name)
		assert.False(t, updated.Active)
		assert.Equal(t, 7, updated.EstimatedMinDays)
		assert.Equal(t, 10, updated.EstimatedMaxDays)
	})

	t.Run("missing method returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/methods/missing",
			`{"name":"Economy"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/methods/"+method.ID, `{`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid day range returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		method := env.seedMethod(t, true)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/methods/"+method.ID,
			`{"name":"Economy","estimated_min_days":5,"estimated_max_days":2}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
