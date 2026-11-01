package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListNotifications(t *testing.T) {
	mux, repo := newTestMux(t)
	seedNotification(t, repo, "n-1", "o-1", "u-1")
	seedNotification(t, repo, "n-2", "o-2", "u-1")
	seedNotification(t, repo, "n-3", "o-3", "u-2")

	t.Run("returns the user's notifications", func(t *testing.T) {
		rec := do(t, mux, "/v1/notifications?user_id=u-1")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data struct {
				Data []notificationResponse `json:"data"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data.Data, 2)
		for _, n := range env.Data.Data {
			assert.Equal(t, "u-1", n.UserID)
		}
	})

	t.Run("unknown user returns an empty list", func(t *testing.T) {
		rec := doAs(t, mux, http.MethodGet, "/v1/notifications", "", "nobody")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data struct {
				Data []notificationResponse `json:"data"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Empty(t, env.Data.Data)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		rec := doUnauth(t, mux, "/v1/notifications")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
