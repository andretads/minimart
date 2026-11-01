package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRate(t *testing.T) {
	t.Run("valid request returns 201", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/rates",
			`{"zone_id":"zone-1","method_id":"method-1","base_cost_cents":599,"free_threshold_cents":5000,"active":true}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var rate ShippingRateResponse
		require.NoError(t, json.Unmarshal(envl.Data, &rate))
		assert.NotEmpty(t, rate.ID)
		assert.Equal(t, "zone-1", rate.ZoneID)
		assert.Equal(t, "method-1", rate.MethodID)
		assert.Equal(t, int64(599), rate.BaseCostCents)
		assert.True(t, rate.Active)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/rates", `{"zone_id":`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing identifiers returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/rates",
			`{"base_cost_cents":100,"active":true}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("negative cost returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/rates",
			`{"zone_id":"zone-1","method_id":"method-1","base_cost_cents":-1,"active":true}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUpdateRate(t *testing.T) {
	t.Run("valid request updates the rate", func(t *testing.T) {
		env := newTestEnv(t)
		rate := env.seedRate(t, "zone-1", "method-1", true)

		rec := env.do(t, http.MethodPatch, "/v1/shipping/rates/"+rate.ID,
			`{"base_cost_cents":899,"free_threshold_cents":10000,"active":false}`)
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var updated ShippingRateResponse
		require.NoError(t, json.Unmarshal(envl.Data, &updated))
		assert.Equal(t, rate.ID, updated.ID)
		assert.Equal(t, int64(899), updated.BaseCostCents)
		assert.Equal(t, int64(10000), updated.FreeThresholdCents)
		assert.False(t, updated.Active)
	})

	t.Run("missing rate returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/rates/missing",
			`{"base_cost_cents":899}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rate := env.seedRate(t, "zone-1", "method-1", true)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/rates/"+rate.ID, `{`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
