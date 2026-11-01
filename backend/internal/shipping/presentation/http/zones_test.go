package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateZone(t *testing.T) {
	t.Run("valid request returns 201", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/zones",
			`{"name":"Southeast","country":"BR","state":"SP","city":"Sao Paulo","postal_code_start":"01000-000","postal_code_end":"19999-999"}`)
		require.Equal(t, http.StatusCreated, rec.Code)

		envl := decodeEnvelope(t, rec)
		require.True(t, envl.Success)
		var zone ShippingZoneResponse
		require.NoError(t, json.Unmarshal(envl.Data, &zone))
		assert.NotEmpty(t, zone.ID)
		assert.Equal(t, "Southeast", zone.Name)
		assert.Equal(t, "BR", zone.Country)
		assert.Equal(t, "01000-000", zone.PostalCodeStart)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/zones", `{"name":`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing required fields returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPost, "/v1/shipping/zones", `{"country":"BR"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestUpdateZone(t *testing.T) {
	t.Run("valid request updates the zone", func(t *testing.T) {
		env := newTestEnv(t)
		zone := env.seedZone(t)

		rec := env.do(t, http.MethodPatch, "/v1/shipping/zones/"+zone.ID,
			`{"name":"South","country":"BR","state":"RS","city":"Porto Alegre"}`)
		require.Equal(t, http.StatusOK, rec.Code)

		envl := decodeEnvelope(t, rec)
		var updated ShippingZoneResponse
		require.NoError(t, json.Unmarshal(envl.Data, &updated))
		assert.Equal(t, zone.ID, updated.ID)
		assert.Equal(t, "South", updated.Name)
		assert.Equal(t, "RS", updated.State)
		assert.Equal(t, "Porto Alegre", updated.City)
	})

	t.Run("missing zone returns 404", func(t *testing.T) {
		env := newTestEnv(t)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/zones/missing",
			`{"name":"South","country":"BR"}`)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		env := newTestEnv(t)
		zone := env.seedZone(t)
		rec := env.do(t, http.MethodPatch, "/v1/shipping/zones/"+zone.ID, `{`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
