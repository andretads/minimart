package carthttp

import (
	"minimart/pkg/response"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlers_GetCart(t *testing.T) {
	mux := newTestMux(t)

	t.Run("creates cart if none exists", func(t *testing.T) {
		rec := doRequest(t, mux, http.MethodGet, "/v1/cart?user_id=user-1", nil, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		var r response.Response
		decodeResponse(t, rec, &r)
		assert.True(t, r.Success)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		rec := doRequestUnauth(t, mux, http.MethodGet, "/v1/cart", nil, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
