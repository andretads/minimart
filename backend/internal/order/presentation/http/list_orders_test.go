package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrders(t *testing.T) {
	mux := newTestMux(t)

	place := func(t *testing.T, userID, key string) {
		t.Helper()
		rec := doWithKey(t, mux, http.MethodPost, "/v1/orders",
			`{"user_id":"`+userID+`","items":[{"product_id":"p1","quantity":1}]}`, key)
		require.Equal(t, http.StatusCreated, rec.Code)
	}
	place(t, "u1", "k1")
	place(t, "u1", "k2")
	place(t, "u2", "k3")

	t.Run("returns only the requested user's orders", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/orders?user_id=u1", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data []orderResponse `json:"data"`
			Meta struct {
				Page       int `json:"page"`
				PageSize   int `json:"page_size"`
				Total      int `json:"total"`
				TotalPages int `json:"total_pages"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data, 2)
		assert.Equal(t, "u1", env.Data[0].UserID)
		assert.Equal(t, "u1", env.Data[1].UserID)
		assert.Equal(t, 1, env.Meta.Page)
		assert.Equal(t, 20, env.Meta.PageSize)
		assert.Equal(t, 2, env.Meta.Total)
		assert.Equal(t, 1, env.Meta.TotalPages)
	})

	t.Run("paginates", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/orders?user_id=u1&page=2&page_size=1", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data []orderResponse `json:"data"`
			Meta struct {
				Page       int `json:"page"`
				PageSize   int `json:"page_size"`
				Total      int `json:"total"`
				TotalPages int `json:"total_pages"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		require.Len(t, env.Data, 1)
		assert.Equal(t, 2, env.Meta.Page)
		assert.Equal(t, 1, env.Meta.PageSize)
		assert.Equal(t, 2, env.Meta.Total)
		assert.Equal(t, 2, env.Meta.TotalPages)
	})

	t.Run("empty list for unknown user", func(t *testing.T) {
		rec := do(t, mux, http.MethodGet, "/v1/orders?user_id=missing", "")
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Data []orderResponse `json:"data"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.Empty(t, env.Data)
		assert.Zero(t, env.Meta.Total)
	})

	badRequests := []struct {
		name string
		path string
	}{
		{name: "missing user id", path: "/v1/orders"},
		{name: "invalid page", path: "/v1/orders?user_id=u1&page=0"},
		{name: "invalid page size", path: "/v1/orders?user_id=u1&page_size=101"},
		{name: "non-numeric page", path: "/v1/orders?user_id=u1&page=abc"},
	}
	for _, tc := range badRequests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, http.MethodGet, tc.path, "")
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
