package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePreferences(t *testing.T) {
	const fullBody = `{"email_enabled":false,"sms_enabled":true,"push_enabled":false,"marketing_enabled":true}`

	t.Run("stores and returns the switches", func(t *testing.T) {
		mux, repo := newTestMux(t)

		rec := doAs(t, mux, http.MethodPut, "/v1/notifications/preferences", fullBody, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data preferencesResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Data.EmailEnabled)
		assert.True(t, env.Data.SMSEnabled)
		assert.False(t, env.Data.PushEnabled)
		assert.True(t, env.Data.MarketingEnabled)
		require.NotNil(t, env.Data.UpdatedAt)

		stored, err := repo.GetByUserID(t.Context(), "u-1")
		require.NoError(t, err)
		assert.False(t, stored.EmailEnabled)
		assert.True(t, stored.MarketingEnabled)
	})

	t.Run("incomplete body returns 400", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodPut, "/v1/notifications/preferences", `{"email_enabled":true}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("malformed body returns 400", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodPut, "/v1/notifications/preferences", "not json", "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux, repo := newTestMux(t)

		rec := doAs(t, mux, http.MethodPut, "/v1/notifications/preferences", fullBody, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)

		_, err := repo.GetByUserID(t.Context(), "u-1")
		require.Error(t, err, "an unauthenticated request must not write preferences")
	})
}
