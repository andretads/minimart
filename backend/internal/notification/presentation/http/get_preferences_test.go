package http

import (
	"encoding/json"
	"minimart/internal/notification/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPreferences(t *testing.T) {
	t.Run("defaults when nothing was saved", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodGet, "/v1/notifications/preferences", "", "u-1")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data preferencesResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.True(t, env.Data.EmailEnabled)
		assert.False(t, env.Data.SMSEnabled)
		assert.False(t, env.Data.PushEnabled)
		assert.False(t, env.Data.MarketingEnabled)
		assert.Nil(t, env.Data.UpdatedAt)
	})

	t.Run("stored preferences are returned", func(t *testing.T) {
		mux, repo := newTestMux(t)
		stored, err := domain.NewPreferences("u-1", false, false, true, true)
		require.NoError(t, err)
		require.NoError(t, repo.Upsert(t.Context(), stored))

		rec := doAs(t, mux, http.MethodGet, "/v1/notifications/preferences", "", "u-1")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data preferencesResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.False(t, env.Data.EmailEnabled)
		assert.False(t, env.Data.SMSEnabled)
		assert.True(t, env.Data.PushEnabled)
		assert.True(t, env.Data.MarketingEnabled)
		require.NotNil(t, env.Data.UpdatedAt)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		mux, _ := newTestMux(t)

		rec := doAs(t, mux, http.MethodGet, "/v1/notifications/preferences", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
